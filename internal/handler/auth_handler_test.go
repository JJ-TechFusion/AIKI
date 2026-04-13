package handler

import (
	"aiki/internal/database"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aiki/internal/config"
	"aiki/internal/domain"
	"aiki/internal/pkg/validator"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	mock.Mock
}

type noopEmailVerificationService struct{}
type noopPasswordResetService struct{}

func (noopEmailVerificationService) SendVerification(ctx context.Context, userID int32) (*domain.EmailVerificationResponse, error) {
	return &domain.EmailVerificationResponse{
		ExpiresInSeconds: 600,
		ResendInSeconds:  60,
	}, nil
}

func (noopEmailVerificationService) VerifyEmail(ctx context.Context, userID int32, otp string) error {
	return nil
}

func (noopPasswordResetService) SendResetCode(ctx context.Context, email string) (*domain.ForgotPasswordResponse, error) {
	return &domain.ForgotPasswordResponse{SessionID: "session-id"}, nil
}

func (noopPasswordResetService) VerifyResetCode(ctx context.Context, sessionID, otp string) error {
	return nil
}

func (noopPasswordResetService) GetVerifiedEmail(ctx context.Context, sessionID string) (string, error) {
	return "john@example.com", nil
}

func (noopPasswordResetService) ClearSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *MockAuthService) ResetPassword(ctx context.Context, email, newPassword string) error {
	//TODO implement me
	return nil
}

func (m *MockAuthService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ForgottenPassword(ctx context.Context, req *domain.ForgotPasswordRequest) (string, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) LinkedInLogin(ctx context.Context, linkedInID, email, firstName, lastName string) (*domain.AuthResponse, error) {
	args := m.Called(ctx, linkedInID, email, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = validator.New()
	return e
}

func TestAuthHandler_Register(t *testing.T) {
	e := setupEcho()
	mockService := new(MockAuthService)
	//cfg := new(config.Config)
	redisCache, _ := database.NewRedisClient(&config.RedisConfig{
		Host:     "127.0.0.1",
		Password: "",
		DB:       0,
		Port:     "6379",
	})
	handler := NewAuthHandler(
		mockService,
		noopEmailVerificationService{},
		noopPasswordResetService{},
		e.Validator,
		redisCache,
		config.Config{},
	)

	t.Run("successful registration", func(t *testing.T) {
		reqBody := domain.RegisterRequest{
			Email:    "john@example.com",
			Password: "Password123!",
		}

		authResp := &domain.AuthResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			User: &domain.User{
				ID:    1,
				Email: reqBody.Email,
			},
		}

		mockService.On("Register", mock.Anything, mock.AnythingOfType("*domain.RegisterRequest")).
			Return(authResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Register(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Register(c)

		require.NoError(t, err) // Echo error handler handles this
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		reqBody := domain.RegisterRequest{
			// Missing required fields
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Register(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	e := setupEcho()
	mockService := new(MockAuthService)
	redisCache, _ := database.NewRedisClient(&config.RedisConfig{
		Host:     "127.0.0.1",
		Password: "",
		DB:       0,
		Port:     "6379",
	})
	handler := NewAuthHandler(
		mockService,
		noopEmailVerificationService{},
		noopPasswordResetService{},
		e.Validator,
		redisCache,
		config.Config{},
	)

	t.Run("successful login", func(t *testing.T) {
		reqBody := domain.LoginRequest{
			Email:    "john@example.com",
			Password: "Password123!",
		}

		authResp := &domain.AuthResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			User: &domain.User{
				ID:    1,
				Email: reqBody.Email,
			},
		}

		mockService.On("Login", mock.Anything, mock.AnythingOfType("*domain.LoginRequest")).
			Return(authResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		reqBody := domain.LoginRequest{
			Email:    "john@example.com",
			Password: "WrongPassword!",
		}

		mockService.On("Login", mock.Anything, mock.AnythingOfType("*domain.LoginRequest")).
			Return(nil, domain.ErrInvalidCredentials).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	e := setupEcho()
	mockService := new(MockAuthService)
	redisCache, _ := database.NewRedisClient(&config.RedisConfig{
		Host:     "127.0.0.1",
		Password: "",
		DB:       0,
		Port:     "6379",
	})
	handler := NewAuthHandler(
		mockService,
		noopEmailVerificationService{},
		noopPasswordResetService{},
		e.Validator,
		redisCache,
		config.Config{},
	)

	t.Run("successful token refresh", func(t *testing.T) {
		reqBody := domain.RefreshTokenRequest{
			RefreshToken: "valid-refresh-token",
		}

		authResp := &domain.AuthResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
		}

		mockService.On("RefreshToken", mock.Anything, reqBody.RefreshToken).
			Return(authResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.RefreshToken(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	e := setupEcho()
	mockService := new(MockAuthService)
	redisCache, _ := database.NewRedisClient(&config.RedisConfig{
		Host:     "127.0.0.1",
		Password: "",
		DB:       0,
		Port:     "6379",
	})
	handler := NewAuthHandler(
		mockService,
		noopEmailVerificationService{},
		noopPasswordResetService{},
		e.Validator,
		redisCache,
		config.Config{},
	)

	t.Run("successful logout", func(t *testing.T) {
		reqBody := domain.RefreshTokenRequest{
			RefreshToken: "valid-refresh-token",
		}

		mockService.On("Logout", mock.Anything, reqBody.RefreshToken).Return(nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Logout(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})
}
