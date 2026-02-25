package service

import (
	"context"
	"testing"
	"time"

	"aiki/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var passwd = "should-not-be-returned"

func TestUserService_GetByID(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		userID := int32(1)
		expectedUser := &domain.User{
			ID: userID,

			Email:        "john@example.com",
			PasswordHash: &passwd,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mockRepo.On("GetByID", ctx, userID).Return(expectedUser, nil).Once()

		user, err := service.GetByID(ctx, userID)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.Email, user.Email)
		assert.Empty(t, user.PasswordHash) // Should not return password hash
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int32(999)

		mockRepo.On("GetByID", ctx, userID).Return(nil, domain.ErrUserNotFound).Once()

		user, err := service.GetByID(ctx, userID)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_GetByEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		email := "john@example.com"
		expectedUser := &domain.User{
			ID: 1,

			Email:        email,
			PasswordHash: &passwd,
			IsActive:     true,
		}

		mockRepo.On("GetByEmail", ctx, email).Return(expectedUser, nil).Once()

		user, err := service.GetByEmail(ctx, email)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Empty(t, user.PasswordHash)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		email := "nonexistent@example.com"

		mockRepo.On("GetByEmail", ctx, email).Return(nil, domain.ErrUserNotFound).Once()

		user, err := service.GetByEmail(ctx, email)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_Update(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		userID := int32(1)
		newFirstName := "Jane"
		newPhone := "+1234567890"

		req := &domain.UpdateUserRequest{
			FirstName:   &newFirstName,
			PhoneNumber: &newPhone,
		}

		updatedUser := &domain.User{
			ID: userID,

			Email:        "jane@example.com",
			PhoneNumber:  &newPhone,
			PasswordHash: &passwd,
			IsActive:     true,
		}

		mockRepo.On("Update", ctx, userID, req.FirstName, req.LastName, req.PhoneNumber).
			Return(updatedUser, nil).Once()

		user, err := service.Update(ctx, userID, req)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, newFirstName, user.FirstName)
		assert.Equal(t, &newPhone, user.PhoneNumber)
		assert.Empty(t, user.PasswordHash)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int32(999)
		req := &domain.UpdateUserRequest{}

		mockRepo.On("Update", ctx, userID, req.FirstName, req.LastName, req.PhoneNumber).
			Return(nil, domain.ErrUserNotFound).Once()

		user, err := service.Update(ctx, userID, req)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}
