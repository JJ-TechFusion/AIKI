package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"aiki/internal/domain"
	"aiki/internal/pkg/jwt"
	"aiki/internal/pkg/password"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) UpdateUserPassword(ctx context.Context, userID int32, newPasswordHash string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockUserRepository) CreateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string, goals []string) (*domain.UserProfile, error) {
	args := m.Called(ctx, userId, fullName, currentJob, experienceLevel, goals)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserRepository) UpdateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string, goals []string) (*domain.UserProfile, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockUserRepository) GetUserProfileByID(ctx context.Context, userId int32) (*domain.UserProfile, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserRepository) UpdateUserJobSearchLocation(ctx context.Context, userID int32, jobSearchLocation string) error {
	return nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User, passwordHash string) (*domain.User, error) {
	args := m.Called(ctx, user, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int32) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, id int32, firstName, lastName *string, phoneNumber *string) (*domain.User, error) {
	args := m.Called(ctx, id, firstName, lastName, phoneNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) IsEmailVerified(ctx context.Context, userID int32) (bool, error) {
	return false, nil
}

func (m *MockUserRepository) MarkEmailVerified(ctx context.Context, userID int32) error {
	return nil
}

func (m *MockUserRepository) CreateRefreshToken(ctx context.Context, userID int32, token string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, token, mock.Anything)
	return args.Error(0)
}

func (m *MockUserRepository) GetRefreshToken(ctx context.Context, token string) (int32, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(int32), args.Error(1)
}

func (m *MockUserRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUserRefreshTokens(ctx context.Context, userID int32) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *MockUserRepository) UploadCV(ctx context.Context, userID int32, data []byte) error {
	args := m.Called(ctx, userID, data)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserCV(ctx context.Context, userID int32) ([]byte, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockUserRepository) GetByLinkedInID(ctx context.Context, linkedInID string) (*domain.User, error) {
	args := m.Called(ctx, linkedInID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) CreateLinkedInUser(ctx context.Context, email, linkedInID string, firstName, lastName *string) (*domain.User, error) {
	args := m.Called(ctx, email, linkedInID, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateLinkedInID(ctx context.Context, userID int32, linkedInID string, firstName, lastName *string) error {
	args := m.Called(ctx, userID, linkedInID, firstName, lastName)
	return args.Error(0)
}

func TestAuthService_Register(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		req := &domain.RegisterRequest{

			Email:    "john@example.com",
			Password: "Password123!",
		}

		expectedUser := &domain.User{
			ID: 1,

			Email:     req.Email,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("EmailExists", ctx, req.Email).Return(false, nil).Once()
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User"), mock.AnythingOfType("string")).
			Return(expectedUser, nil).Once()
		mockRepo.On("CreateRefreshToken", ctx, expectedUser.ID, mock.AnythingOfType("string"), mock.Anything).
			Return(nil).Once()

		resp, err := service.Register(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, expectedUser.Email, resp.User.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("weak password", func(t *testing.T) {
		req := &domain.RegisterRequest{

			Email:    "john@example.com",
			Password: "weak",
		}

		_, err := service.Register(ctx, req)

		assert.Error(t, err)
	})

	t.Run("email already exists", func(t *testing.T) {
		req := &domain.RegisterRequest{

			Email:    "existing@example.com",
			Password: "Password123!",
		}

		mockRepo.On("EmailExists", ctx, req.Email).Return(true, nil).Once()

		_, err := service.Register(ctx, req)

		assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	t.Run("successful login", func(t *testing.T) {
		req := &domain.LoginRequest{
			Email:    "john@example.com",
			Password: "Password123!",
		}

		hashedPassword, _ := password.Hash(req.Password)
		existingUser := &domain.User{
			ID: 1,

			Email:        req.Email,
			PasswordHash: &hashedPassword,
			IsActive:     true,
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(existingUser, nil).Once()
		mockRepo.On("DeleteUserRefreshTokens", ctx, existingUser.ID).Return(nil).Once()
		mockRepo.On("CreateRefreshToken", ctx, existingUser.ID, mock.AnythingOfType("string"), mock.Anything).
			Return(nil).Once()

		resp, err := service.Login(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, existingUser.Email, resp.User.Email)
		assert.Empty(t, resp.User.PasswordHash) // Should not return password hash
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		req := &domain.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "Password123!",
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, domain.ErrUserNotFound).Once()

		_, err := service.Login(ctx, req)

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		req := &domain.LoginRequest{
			Email:    "john@example.com",
			Password: "WrongPassword123!",
		}

		hashedPassword, _ := password.Hash("CorrectPassword123!")
		existingUser := &domain.User{
			ID:           1,
			Email:        req.Email,
			PasswordHash: &hashedPassword,
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(existingUser, nil).Once()

		_, err := service.Login(ctx, req)

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_LinkedInLogin_CreatesMissingProfile(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	firstName := "Jane"
	lastName := "Doe"
	email := "jane@example.com"
	linkedInID := "linkedin-123"
	fullName := "Jane Doe"

	linkedInUser := &domain.User{
		ID:         7,
		Email:      email,
		FirstName:  &firstName,
		LastName:   &lastName,
		LinkedInID: &linkedInID,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	createdProfile := &domain.UserProfile{
		UserId:   linkedInUser.ID,
		FullName: fullName,
	}

	mockRepo.On("GetByLinkedInID", ctx, linkedInID).
		Return(linkedInUser, nil).Once()
	mockRepo.On("GetUserProfileByID", ctx, linkedInUser.ID).
		Return(nil, domain.ErrUserNotFound).Once()
	mockRepo.On("CreateUserProfile", ctx, linkedInUser.ID, &fullName, (*string)(nil), (*string)(nil), []string(nil)).
		Return(createdProfile, nil).Once()
	mockRepo.On("DeleteUserRefreshTokens", ctx, linkedInUser.ID).
		Return(nil).Once()
	mockRepo.On("CreateRefreshToken", ctx, linkedInUser.ID, mock.AnythingOfType("string"), mock.Anything).
		Return(nil).Once()

	resp, err := service.LinkedInLogin(ctx, linkedInID, email, firstName, lastName)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, linkedInUser.Email, resp.User.Email)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_RefreshToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	t.Run("successful token refresh", func(t *testing.T) {
		refreshToken := "valid-refresh-token"
		userID := int32(1)
		user := &domain.User{
			ID: userID,

			Email:    "john@example.com",
			IsActive: true,
		}

		mockRepo.On("GetRefreshToken", ctx, refreshToken).Return(userID, nil).Once()
		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Once()
		mockRepo.On("DeleteRefreshToken", ctx, refreshToken).Return(nil).Once()
		mockRepo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.Anything).
			Return(nil).Once()

		resp, err := service.RefreshToken(ctx, refreshToken)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.NotEqual(t, refreshToken, resp.RefreshToken) // Should be a new token
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		refreshToken := "invalid-token"

		mockRepo.On("GetRefreshToken", ctx, refreshToken).Return(int32(0), domain.ErrInvalidToken).Once()

		_, err := service.RefreshToken(ctx, refreshToken)

		assert.ErrorIs(t, err, domain.ErrInvalidToken)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Logout(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	refreshToken := "valid-refresh-token"
	mockRepo.On("DeleteRefreshToken", ctx, refreshToken).Return(nil).Once()

	err := service.Logout(ctx, refreshToken)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Logout_Error(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	service := NewAuthService(mockRepo, jwtManager)
	ctx := context.Background()

	refreshToken := "token-with-error"
	expectedError := errors.New("database error")
	mockRepo.On("DeleteRefreshToken", ctx, refreshToken).Return(expectedError).Once()

	err := service.Logout(ctx, refreshToken)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}
