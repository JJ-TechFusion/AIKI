package service

import (
	"aiki/internal/domain"
	"aiki/internal/repository"
	"aiki/internal/serp"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

type SerpJobService interface {
	GetJobsForUser(ctx context.Context, userID int32, location string) (*domain.JobSearchResult, error)
	SaveJobToTracker(ctx context.Context, userID int32, cacheID int32) (*domain.Job, error)
	ApplyRecommendedJob(ctx context.Context, userID int32, cacheID int32, notes string) (*domain.DirectApplyResult, error)
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

func (s *serpJobService) GetJobsForUser(ctx context.Context, userID int32, location string) (*domain.JobSearchResult, error) {
	loc := strings.TrimSpace(location)

	profile, err := s.userRepo.GetUserProfileByID(ctx, userID)
	if err != nil {
		return nil, errors.New("please complete your profile (job title and experience level) before searching for jobs")
	}
	if profile.CurrentJob == "" {
		return nil, errors.New("please add your current job title to your profile to get job recommendations")
	}

	// Return from cache if fresh and location filter matches the last successful search
	lastFetch, err := s.serpRepo.GetLatestFetchTime(ctx, userID)
	if err == nil && lastFetch != nil && time.Since(*lastFetch) < cacheTTL && profile.JobSearchLocation == loc {
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

	// Delete stale cache before a new Serp fetch
	_ = s.serpRepo.DeleteOldCache(ctx, userID)

	jobs, err := s.serpClient.FetchJobs(profile.CurrentJob, profile.ExperienceLevel, loc)
	if err != nil {
		log.Printf("serp api fetch failed for user %d: %v", userID, err)
		if profile.JobSearchLocation == loc {
			cached, cacheErr := s.serpRepo.GetCachedJobs(ctx, userID, 20, 0)
			if cacheErr == nil && len(cached) > 0 {
				return &domain.JobSearchResult{
					Jobs:       cached,
					TotalCount: len(cached),
					FromCache:  true,
					FetchedAt:  cached[0].FetchedAt,
				}, nil
			}
		}
		return nil, errors.New("failed to fetch jobs, please try again later")
	}

	if len(jobs) == 0 {
		if err := s.userRepo.UpdateUserJobSearchLocation(ctx, userID, loc); err != nil {
			log.Printf("failed to persist job search location for user %d: %v", userID, err)
		}
		return &domain.JobSearchResult{
			Jobs:       []domain.SerpJobCache{},
			TotalCount: 0,
			FromCache:  false,
			FetchedAt:  time.Now(),
		}, nil
	}

	cached, err := s.serpRepo.UpsertJobs(ctx, userID, jobs)
	if err != nil {
		log.Printf("failed to cache serp jobs for user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to cache job listings: %w", err)
	}

	if err := s.userRepo.UpdateUserJobSearchLocation(ctx, userID, loc); err != nil {
		log.Printf("failed to persist job search location for user %d: %v", userID, err)
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
		return nil, err
	}

	if cached.SavedToTracker {
		return nil, domain.ErrJobAlreadyTracked
	}

	newJob := &domain.Job{
		UserId:      userID,
		Title:       cached.Title,
		CompanyName: cached.CompanyName,
		Location:    cached.Location,
		Link:        cached.Link,
		Platform:    cached.Platform,
		Status:      domain.JobStatusSaved,
	}

	jobID, err := s.jobRepo.Create(ctx, newJob)
	if err != nil {
		return nil, err
	}

	newJob.ID = jobID

	if err := s.serpRepo.MarkSavedToTracker(ctx, cacheID, userID, jobID); err != nil {
		return nil, err
	}

	return s.jobRepo.GetJobByID(ctx, jobID)
}

func (s *serpJobService) ApplyRecommendedJob(ctx context.Context, userID int32, cacheID int32, notes string) (*domain.DirectApplyResult, error) {
	cached, err := s.serpRepo.GetCachedJobByID(ctx, cacheID, userID)
	if err != nil {
		return nil, err
	}

	applyURL := cached.Link
	if applyURL == "" {
		return nil, domain.ErrNoApplyLink
	}

	if cached.TrackerJobID != nil && *cached.TrackerJobID > 0 {
		existing, err := s.jobRepo.GetJobByID(ctx, *cached.TrackerJobID)
		if err != nil {
			return nil, err
		}
		if existing.UserId != userID {
			return nil, domain.ErrUnauthorized
		}
		if existing.Status == domain.JobStatusApplied {
			return &domain.DirectApplyResult{Job: *existing, ApplyURL: applyURL}, nil
		}
		existing.Status = domain.JobStatusApplied
		existing.DateApplied = time.Now().Format("2006-01-02")
		if notes != "" {
			if existing.Notes != "" {
				existing.Notes = existing.Notes + "\n" + notes
			} else {
				existing.Notes = notes
			}
		}
		if err := s.jobRepo.Update(ctx, existing.ID, existing); err != nil {
			return nil, err
		}
		updated, err := s.jobRepo.GetJobByID(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		return &domain.DirectApplyResult{Job: *updated, ApplyURL: applyURL}, nil
	}

	newJob := &domain.Job{
		UserId:      userID,
		Title:       cached.Title,
		CompanyName: cached.CompanyName,
		Location:    cached.Location,
		Link:        cached.Link,
		Platform:    cached.Platform,
		Status:      domain.JobStatusApplied,
		Notes:       notes,
		DateApplied: time.Now().Format("2006-01-02"),
	}

	jobID, err := s.jobRepo.Create(ctx, newJob)
	if err != nil {
		return nil, err
	}

	if err := s.serpRepo.MarkSavedToTracker(ctx, cacheID, userID, jobID); err != nil {
		return nil, err
	}

	newJob.ID = jobID
	full, err := s.jobRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return &domain.DirectApplyResult{Job: *full, ApplyURL: applyURL}, nil
}
