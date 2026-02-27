package service

import (
	"aiki/internal/domain"
	"aiki/internal/repository"
	"aiki/internal/serp"
	"context"
	"errors"
	"log"
	"time"
)

const cacheTTL = 24 * time.Hour

type SerpJobService interface {
	GetJobsForUser(ctx context.Context, userID int32) (*domain.JobSearchResult, error)
	SaveJobToTracker(ctx context.Context, userID int32, cacheID int32) (*domain.Job, error)
}

type serpJobService struct {
	serpRepo   repository.SerpJobRepository
	userRepo   repository.UserRepository
	jobRepo    repository.JobRepository
	serpClient *serp.Client
}

func NewSerpJobService(
	serpRepo repository.SerpJobRepository,
	userRepo repository.UserRepository,
	jobRepo repository.JobRepository,
	serpClient *serp.Client,
) SerpJobService {
	return &serpJobService{
		serpRepo:   serpRepo,
		userRepo:   userRepo,
		jobRepo:    jobRepo,
		serpClient: serpClient,
	}
}

func (s *serpJobService) GetJobsForUser(ctx context.Context, userID int32) (*domain.JobSearchResult, error) {
	// Return from cache if fresh
	lastFetch, err := s.serpRepo.GetLatestFetchTime(ctx, userID)
	if err == nil && lastFetch != nil && time.Since(*lastFetch) < cacheTTL {
		cached, err := s.serpRepo.GetCachedJobs(ctx, userID, 20, 0)
		if err != nil {
			return nil, err
		}
		return &domain.JobSearchResult{
			Jobs:       cached,
			TotalCount: len(cached),
			FromCache:  true,
			FetchedAt:  *lastFetch,
		}, nil
	}

	// Get user profile for search query
	profile, err := s.userRepo.GetUserProfileByID(ctx, userID)
	if err != nil {
		return nil, errors.New("please complete your profile (job title and experience level) before searching for jobs")
	}
	if profile.CurrentJob == "" {
		return nil, errors.New("please add your current job title to your profile to get job recommendations")
	}

	// Delete stale cache
	_ = s.serpRepo.DeleteOldCache(ctx, userID)

	// Fetch from SerpApi
	jobs, err := s.serpClient.FetchJobs(profile.CurrentJob, profile.ExperienceLevel, "")
	if err != nil {
		log.Printf("serp api fetch failed for user %d: %v", userID, err)
		// Fall back to stale cache
		cached, cacheErr := s.serpRepo.GetCachedJobs(ctx, userID, 20, 0)
		if cacheErr == nil && len(cached) > 0 {
			return &domain.JobSearchResult{
				Jobs:       cached,
				TotalCount: len(cached),
				FromCache:  true,
				FetchedAt:  cached[0].FetchedAt,
			}, nil
		}
		return nil, errors.New("failed to fetch jobs, please try again later")
	}

	if len(jobs) == 0 {
		return &domain.JobSearchResult{
			Jobs:       []domain.SerpJobCache{},
			TotalCount: 0,
			FromCache:  false,
			FetchedAt:  time.Now(),
		}, nil
	}

	// Store in cache
	cached, err := s.serpRepo.UpsertJobs(ctx, userID, jobs)
	if err != nil {
		log.Printf("failed to cache serp jobs for user %d: %v", userID, err)
		cached = make([]domain.SerpJobCache, len(jobs))
		for i, j := range jobs {
			cached[i] = domain.SerpJobCache{
				UserID:      userID,
				ExternalID:  j.ExternalID,
				Title:       j.Title,
				CompanyName: j.CompanyName,
				Location:    j.Location,
				Description: j.Description,
				Link:        j.Link,
				Platform:    j.Platform,
				PostedAt:    j.PostedAt,
				Salary:      j.Salary,
				FetchedAt:   j.FetchedAt,
			}
		}
	}

	return &domain.JobSearchResult{
		Jobs:       cached,
		TotalCount: len(cached),
		FromCache:  false,
		FetchedAt:  time.Now(),
	}, nil
}

func (s *serpJobService) SaveJobToTracker(ctx context.Context, userID int32, cacheID int32) (*domain.Job, error) {
	cached, err := s.serpRepo.GetCachedJobByID(ctx, cacheID, userID)
	if err != nil {
		return nil, errors.New("job not found")
	}

	if cached.SavedToTracker {
		return nil, errors.New("job already saved to tracker")
	}

	// Build the job — jobRepo.Create returns int32 (the new job ID)
	newJob := &domain.Job{
		UserId:      userID,
		Title:       cached.Title,
		CompanyName: cached.CompanyName,
		Location:    cached.Location,
		Link:        cached.Link,
		Platform:    cached.Platform,
		Status:      "applied",
		DateApplied: time.Now().Format("2006-01-02"),
	}

	jobID, err := s.jobRepo.Create(ctx, newJob)
	if err != nil {
		return nil, err
	}

	newJob.ID = jobID

	// Mark as saved in the cache
	_ = s.serpRepo.MarkSavedToTracker(ctx, cacheID, userID)

	return newJob, nil
}
