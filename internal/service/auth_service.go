package service

import (
	"context"
	"errors"
	"fmt"
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
	ForgottenPassword(ctx context.Context, req *domain.ForgotPasswordRequest) (string, error)
	ResetPassword(ctx context.Context, userEmail, newPassword string) error
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

func (s *authService) ForgottenPassword(ctx context.Context, req *domain.ForgotPasswordRequest) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return "", err
	}
	// TODO: implement sending token to email
	return user.Email, nil
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
