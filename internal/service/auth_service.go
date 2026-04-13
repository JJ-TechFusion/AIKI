package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"aiki/internal/domain"
	"aiki/internal/pkg/jwt"
	"aiki/internal/pkg/password"
	"aiki/internal/repository"
)

//go:generate mockgen -source=auth_service.go -destination=mocks/mock_auth_service.go -package=mocks

type AuthService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ResetPassword(ctx context.Context, userEmail, newPassword string) error
	LinkedInLogin(ctx context.Context, linkedInID, email, firstName, lastName string) (*domain.AuthResponse, error)
}

type authService struct {
	userRepo   repository.UserRepository
	jwtManager *jwt.Manager
}

func NewAuthService(userRepo repository.UserRepository, jwtManager *jwt.Manager) AuthService {
	return &authService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *authService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	if err := password.Validate(req.Password); err != nil {
		return nil, err
	}

	exists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrEmailAlreadyExists
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email: req.Email,
	}

	createdUser, err := s.userRepo.Create(ctx, user, hashedPassword)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(createdUser.ID, createdUser.Email)
	if err != nil {
		return nil, err
	}

	refreshToken := s.jwtManager.GenerateRefreshToken()
	expiresAt := time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())

	// Store refresh token
	if err := s.userRepo.CreateRefreshToken(ctx, createdUser.ID, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         createdUser,
	}, nil
}

func (s *authService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if err := password.Compare(*user.PasswordHash, req.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken := s.jwtManager.GenerateRefreshToken()
	expiresAt := time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())

	// Store refresh token (delete old ones first)
	if err := s.userRepo.DeleteUserRefreshTokens(ctx, user.ID); err != nil {
		return nil, err
	}

	if err := s.userRepo.CreateRefreshToken(ctx, user.ID, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	// Validate refresh token
	userID, err := s.userRepo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token
	newRefreshToken := s.jwtManager.GenerateRefreshToken()
	expiresAt := time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())

	// Delete old refresh token and create new one
	if err := s.userRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	if err := s.userRepo.CreateRefreshToken(ctx, user.ID, newRefreshToken, expiresAt); err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.userRepo.DeleteRefreshToken(ctx, refreshToken)
}

func (s *authService) ResetPassword(ctx context.Context, userEmail, newPassword string) error {
	// get user by email and then update the user password
	user, err := s.userRepo.GetByEmail(ctx, userEmail)
	if err != nil {
		return err
	}
	hashPassword, err := password.Hash(newPassword)
	if err != nil {
		fmt.Println("error hashing password")
		return err
	}
	if err = s.userRepo.UpdateUserPassword(ctx, user.ID, hashPassword); err != nil {
		fmt.Println("error updating user password: ", err)
		return errors.New("error updating user password, please try again")
	}
	return nil
}

func (s *authService) LinkedInLogin(ctx context.Context, linkedInID, email, firstName, lastName string) (*domain.AuthResponse, error) {
	// Try to find existing user by LinkedIn ID
	user, err := s.userRepo.GetByLinkedInID(ctx, linkedInID)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, err
	}

	// If not found by LinkedIn ID, try by email
	if user == nil {
		user, err = s.userRepo.GetByEmail(ctx, email)
		if err != nil && err != domain.ErrUserNotFound {
			return nil, err
		}
	}

	if user != nil {
		// Link LinkedIn ID to existing user if not already linked
		if user.LinkedInID == nil || *user.LinkedInID != linkedInID {
			if err := s.userRepo.UpdateLinkedInID(ctx, user.ID, linkedInID, &firstName, &lastName); err != nil {
				return nil, err
			}
		}
	} else {
		// Create new user with LinkedIn info
		user, err = s.userRepo.CreateLinkedInUser(ctx, email, linkedInID, &firstName, &lastName)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureUserProfileForLinkedIn(ctx, user.ID, firstName, lastName); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken := s.jwtManager.GenerateRefreshToken()
	expiresAt := time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())

	// Delete old refresh tokens and create new one
	_ = s.userRepo.DeleteUserRefreshTokens(ctx, user.ID)
	if err := s.userRepo.CreateRefreshToken(ctx, user.ID, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *authService) ensureUserProfileForLinkedIn(
	ctx context.Context,
	userID int32,
	firstName, lastName string,
) error {
	_, err := s.userRepo.GetUserProfileByID(ctx, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return err
	}

	fullName := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
	var fullNamePtr *string
	if fullName != "" {
		fullNamePtr = &fullName
	}

	_, err = s.userRepo.CreateUserProfile(ctx, userID, fullNamePtr, nil, nil, nil)
	if errors.Is(err, domain.ErrUserProfileAlreadyExists) {
		return nil
	}

	return err
}
