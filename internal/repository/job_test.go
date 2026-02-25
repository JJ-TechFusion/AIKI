package repository

import (
	"aiki/internal/domain"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobs_Create(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewJobRepository(pool)
	userRepo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{

		Email:       "jane.Doe@test.com",
		PhoneNumber: stringPtr("+234188399439"),
	}
	createdUser, err := userRepo.Create(ctx, user, "hashed_password")
	assert.NotNil(t, createdUser)
	require.NoError(t, err)
	job := &domain.Job{
		UserId:      createdUser.ID,
		Title:       "demo job",
		CompanyName: "test company",
		Location:    "test location",
		Platform:    "test platform", // could be LinkedIn, website or indeed or others
		Link:        "test link",
		Status:      "applied",
		DateApplied: "2025-10-12", // yyyy-mm-dd
	}

	createdJob, err := repo.Create(ctx, job)
	require.NoError(t, err)
	assert.NotNil(t, createdJob)
}

func TestJobs_Update(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewJobRepository(pool)
	userRepo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{

		Email:       "john.Doe@test.com",
		PhoneNumber: stringPtr("+234188399439"),
	}
	createdUser, err := userRepo.Create(ctx, user, "hashed_password")
	assert.NotNil(t, createdUser)
	require.NoError(t, err)
	job := &domain.Job{
		UserId:      createdUser.ID,
		Title:       "another job",
		CompanyName: "another test company",
		Location:    "another test location",
		Platform:    "another test platform", // could be LinkedIn, website or indeed or others
		Link:        "testlink.com",
		Status:      "applied",
		DateApplied: "2025-10-12", // yyyy-mm-dd
	}

	jobId, err := repo.Create(ctx, job)
	require.NoError(t, err)
	assert.NotNil(t, jobId)

	updateJob := &domain.Job{
		CompanyName: "wells fargo",
	}

	err = repo.Update(ctx, jobId, updateJob)
	require.NoError(t, err)
}

func TestJob_GetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewJobRepository(pool)
	userRepo := NewUserRepository(pool)
	ctx := context.Background()
	user, err := userRepo.GetByEmail(ctx, "john.Doe@test.com")
	require.NoError(t, err)
	assert.NotNil(t, user)
	job, err := repo.GetJobByID(ctx, 2)
	fmt.Println("Job fetched:", job, err)
	require.NoError(t, err)
	require.NotNil(t, job)
}

func TestJobs_Delete(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewJobRepository(pool)
	userRepo := NewUserRepository(pool)
	ctx := context.Background()
	user := &domain.User{

		Email:       "kiddo.Doe@test.com",
		PhoneNumber: stringPtr("+234188399439"),
	}
	createdUser, err := userRepo.Create(ctx, user, "hashed_password")
	assert.NotNil(t, createdUser)
	require.NoError(t, err)
	job := &domain.Job{
		UserId:      createdUser.ID,
		Title:       "another job",
		CompanyName: "another test company",
		Location:    "another test location",
		Platform:    "another test platform", // could be LinkedIn, website or indeed or others
		Link:        "testlink.com",
		Status:      "applied",
		DateApplied: "2025-10-12", // yyyy-mm-dd
	}
	jobId, err := repo.Create(ctx, job)
	require.NoError(t, err)
	assert.NotNil(t, jobId)

	err = repo.DeleteJob(ctx, jobId)
	require.NoError(t, err)
}
