package service

import (
	"aiki/internal/domain"
	"aiki/internal/repository"
	"context"
)

//go:generate mockgen -source=user_service.go -destination=mocks/mock_user_service.go -package=mocks

type UserService interface {
	GetByID(ctx context.Context, id int32) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, id int32, req *domain.UpdateUserRequest) (*domain.User, error)
	CreateUserProfile(ctx context.Context, userProfile domain.UserProfile) (*domain.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userProfile domain.UserProfile) (*domain.UserProfile, error)
	GetUserProfile(ctx context.Context, id int32) (*domain.UserProfile, error)
	UploadUserCV(ctx context.Context, id int32, data []byte) error
	GetUserCV(ctx context.Context, id int32) ([]byte, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetByID(ctx context.Context, id int32) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil
	return user, nil
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil
	return user, nil
}

func (s *userService) Update(ctx context.Context, id int32, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.Update(ctx, id, req.FirstName, req.LastName, req.PhoneNumber)
	if err != nil {
		return nil, err
	}

	// Don't return password hash
	user.PasswordHash = nil
	return user, nil
}

// ========================================================
// 			user profile
// ========================================================

func (s *userService) CreateUserProfile(ctx context.Context, userProfile domain.UserProfile) (*domain.UserProfile, error) {
	profile, err := s.userRepo.CreateUserProfile(ctx, userProfile.UserId, &userProfile.FullName, &userProfile.CurrentJob, &userProfile.ExperienceLevel, userProfile.Goals)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *userService) UpdateUserProfile(ctx context.Context, user domain.UserProfile) (*domain.UserProfile, error) {
	userProfile, err := s.userRepo.GetUserProfileByID(ctx, user.UserId)
	if err != nil {
		return nil, err
	}
	if &user.FullName != nil && len(user.FullName) > 5 {
		userProfile.FullName = user.FullName
	}
	if &user.CurrentJob != nil && len(user.CurrentJob) > 5 {
		userProfile.CurrentJob = user.CurrentJob
	}
	if &user.ExperienceLevel != nil && len(user.ExperienceLevel) > 5 {
		userProfile.ExperienceLevel = user.ExperienceLevel
	}
	if len(user.Goals) > 0 {
		userProfile.Goals = user.Goals
	}
	profile, err := s.userRepo.UpdateUserProfile(ctx, userProfile.UserId, &userProfile.FullName, &userProfile.CurrentJob, &userProfile.ExperienceLevel, userProfile.Goals)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *userService) GetUserProfile(ctx context.Context, id int32) (*domain.UserProfile, error) {
	profile, err := s.userRepo.GetUserProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *userService) UploadUserCV(ctx context.Context, id int32, data []byte) error {
	err := s.userRepo.UploadCV(ctx, id, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *userService) GetUserCV(ctx context.Context, id int32) ([]byte, error) {
	cv, err := s.userRepo.GetUserCV(ctx, id)
	if err != nil {
		return nil, err
	}
	return cv, nil
}
