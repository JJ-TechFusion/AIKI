package repository

import (
	"context"
	"testing"
	"time"

	"aiki/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests - these require a running PostgreSQL instance
// Run with: docker-compose up -d postgres
// Skip with: go test -short ./...

func setupTestDB(t *testing.T) *pgxpool.Pool {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	connString := "host=localhost port=5433 user=aiki_test password=aiki_test_password dbname=aiki_test_db sslmode=disable"

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	// Clean up test data
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM refresh_tokens")
		pool.Exec(ctx, "DELETE FROM user_profile")
		pool.Exec(context.Background(), "DELETE FROM users WHERE email LIKE '%@test.com'")
		pool.Exec(context.Background(), "DELETE FROM users")
		pool.Exec(ctx, "DELETE FROM jobs")
		pool.Close()
	})

	return pool
}

func TestUserRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{

		Email:       "john.doe@test.com",
		PhoneNumber: stringPtr("+1234567890"),
	}

	createdUser, err := repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)
	assert.NotZero(t, createdUser.ID)
	assert.Equal(t, user.FirstName, createdUser.FirstName)
	assert.Equal(t, user.LastName, createdUser.LastName)
	assert.Equal(t, user.Email, createdUser.Email)
	assert.Equal(t, user.PhoneNumber, createdUser.PhoneNumber)
	assert.True(t, createdUser.IsActive)
	assert.NotZero(t, createdUser.CreatedAt)
}

func TestUserRepository_GetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user first
	user := &domain.User{

		Email: "jane.smith@test.com",
	}
	createdUser, err := repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)

	// Get the user by ID
	foundUser, err := repo.GetByID(ctx, createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, foundUser.ID)
	assert.Equal(t, createdUser.Email, foundUser.Email)

	// Try to get non-existent user
	_, err = repo.GetByID(ctx, 999999)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user first
	user := &domain.User{

		Email: "bob.johnson@test.com",
	}
	createdUser, err := repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)

	// Get the user by email
	foundUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, foundUser.ID)
	assert.Equal(t, createdUser.Email, foundUser.Email)

	// Try to get non-existent user
	_, err = repo.GetByEmail(ctx, "nonexistent@test.com")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user first
	user := &domain.User{

		Email: "alice.williams@test.com",
	}
	createdUser, err := repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)

	// Update the user
	newFirstName := "Alicia"
	newPhoneNumber := "+9876543210"
	updatedUser, err := repo.Update(ctx, createdUser.ID, &newFirstName, nil, &newPhoneNumber)
	require.NoError(t, err)
	assert.Equal(t, newFirstName, updatedUser.FirstName)
	assert.Equal(t, createdUser.LastName, updatedUser.LastName) // Unchanged
	assert.Equal(t, &newPhoneNumber, updatedUser.PhoneNumber)
}

func TestUserRepository_EmailExists(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	email := "exists@test.com"

	// Check before creating
	exists, err := repo.EmailExists(ctx, email)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create a user
	user := &domain.User{

		Email: email,
	}
	_, err = repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)

	// Check after creating
	exists, err = repo.EmailExists(ctx, email)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepository_RefreshTokens(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user first
	user := &domain.User{

		Email: "token.tester@test.com",
	}
	createdUser, err := repo.Create(ctx, user, "hashed_password")
	require.NoError(t, err)

	token := "test-refresh-token-123"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create refresh token
	err = repo.CreateRefreshToken(ctx, createdUser.ID, token, expiresAt)
	require.NoError(t, err)

	// Get refresh token
	userID, err := repo.GetRefreshToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, userID)

	// Delete refresh token
	err = repo.DeleteRefreshToken(ctx, token)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetRefreshToken(ctx, token)
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestUserRepository_CreateUserProfile(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{

		Email:       "john.doe@test.com",
		PhoneNumber: stringPtr("+1234567890"),
	}

	createdUser, err := repo.Create(ctx, user, "hashed_password")
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)

	profile := &domain.UserProfile{
		UserId:          createdUser.ID,
		ExperienceLevel: "beginner",
		CurrentJob:      "backend developer",
		FullName:        "John Smith",
	}
	fullName := stringPtr(profile.FullName)
	xpLevel := stringPtr(profile.ExperienceLevel)
	currentJob := stringPtr(profile.CurrentJob)
	createdProfile, err := repo.CreateUserProfile(ctx, createdUser.ID, fullName, currentJob, xpLevel)
	assert.NoError(t, err)
	assert.NotNil(t, createdProfile)
	assert.Equal(t, createdUser.ID, createdProfile.UserId)
}

func TestUserRepository_GetUserProfile(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()
	user := &domain.User{

		Email:       "test@test.com",
		PhoneNumber: stringPtr("+1234567890"),
	}

	createdUser, err := repo.Create(ctx, user, "hashed_password")
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	pf := &domain.UserProfile{
		UserId:          createdUser.ID,
		FullName:        "John Smith",
		CurrentJob:      "backend developer",
		ExperienceLevel: "beginner",
	}
	createUserProfile, err := repo.CreateUserProfile(ctx, pf.UserId, &pf.FullName, &pf.CurrentJob, &pf.ExperienceLevel)
	assert.NoError(t, err)
	assert.Equal(t, createdUser.ID, createUserProfile.UserId)
	profile, err := repo.GetUserProfileByID(ctx, createdUser.ID)
	assert.NoError(t, err)
	assert.NotNil(t, profile)
}

func stringPtr(s string) *string {
	return &s
}
