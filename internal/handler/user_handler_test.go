package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aiki/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserService is a mock implementation of UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetByID(ctx context.Context, id int32) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, id int32, req *domain.UpdateUserRequest) (*domain.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) CreateUserProfile(ctx context.Context, req domain.UserProfile) (*domain.UserProfile, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserService) GetUserProfile(ctx context.Context, id int32) (*domain.UserProfile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserService) UpdateUserProfile(ctx context.Context, req domain.UserProfile) (*domain.UserProfile, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserService) UploadUserCV(ctx context.Context, userID int32, data []byte) error {
	args := m.Called(ctx, userID, data)
	return args.Error(0)
}

func TestUserHandler_GetMe(t *testing.T) {
	e := setupEcho()
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService, e.Validator)

	t.Run("successful retrieval", func(t *testing.T) {
		userID := int32(1)
		expectedUser := &domain.User{
			ID: userID,

			Email: "john@example.com",
		}

		mockService.On("GetByID", mock.Anything, userID).Return(expectedUser, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", userID) // Simulate auth middleware

		err := handler.GetMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized - missing user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Don't set user_id

		err := handler.GetMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int32(999)

		mockService.On("GetByID", mock.Anything, userID).Return(nil, domain.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", userID)

		err := handler.GetMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_UpdateMe(t *testing.T) {
	e := setupEcho()
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService, e.Validator)

	t.Run("successful update", func(t *testing.T) {
		userID := int32(1)
		newFirstName := "Jane"
		reqBody := domain.UpdateUserRequest{
			FirstName: &newFirstName,
		}

		updatedUser := &domain.User{
			ID: userID,

			Email: "jane@example.com",
		}

		mockService.On("Update", mock.Anything, userID, mock.AnythingOfType("*domain.UpdateUserRequest")).
			Return(updatedUser, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", userID)

		err := handler.UpdateMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized - missing user_id", func(t *testing.T) {
		reqBody := domain.UpdateUserRequest{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		userID := int32(1)
		invalidName := "J" // Too short
		reqBody := domain.UpdateUserRequest{
			FirstName: &invalidName,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", userID)

		err := handler.UpdateMe(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
