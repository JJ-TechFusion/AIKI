package repository

import (
	"aiki/internal/database/db"
	"aiki/internal/domain"
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SerpJobRepository interface {
	UpsertJobs(ctx context.Context, userID int32, jobs []domain.SerpJob) ([]domain.SerpJobCache, error)
	GetCachedJobs(ctx context.Context, userID int32, limit, offset int32) ([]domain.SerpJobCache, error)
	GetCachedJobByID(ctx context.Context, jobID, userID int32) (*domain.SerpJobCache, error)
	GetLatestFetchTime(ctx context.Context, userID int32) (*time.Time, error)
	MarkSavedToTracker(ctx context.Context, jobID, userID int32) error
	DeleteOldCache(ctx context.Context, userID int32) error
}

type serpJobRepository struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func NewSerpJobRepository(dbPool *pgxpool.Pool) SerpJobRepository {
	return &serpJobRepository{db: dbPool, queries: db.New(dbPool)}
}

func (r *serpJobRepository) UpsertJobs(ctx context.Context, userID int32, jobs []domain.SerpJob) ([]domain.SerpJobCache, error) {
	cached := make([]domain.SerpJobCache, 0, len(jobs))
	for _, job := range jobs {
		row, err := r.queries.UpsertSerpJobCache(ctx, db.UpsertSerpJobCacheParams{
			UserID:      userID,
			ExternalID:  job.ExternalID,
			Title:       job.Title,
			CompanyName: nullableString(job.CompanyName),
			Location:    nullableString(job.Location),
			Description: nullableString(job.Description),
			Link:        nullableString(job.Link),
			Platform:    nullableString(job.Platform),
			PostedAt:    nullableString(job.PostedAt),
			Salary:      nullableString(job.Salary),
		})
		if err != nil {
			return nil, err
		}
		cached = append(cached, mapSerpJob(row))
	}
	return cached, nil
}

func (r *serpJobRepository) GetCachedJobs(ctx context.Context, userID int32, limit, offset int32) ([]domain.SerpJobCache, error) {
	rows, err := r.queries.GetCachedJobsByUserID(ctx, db.GetCachedJobsByUserIDParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	jobs := make([]domain.SerpJobCache, len(rows))
	for i, row := range rows {
		jobs[i] = mapSerpJob(row)
	}
	return jobs, nil
}

func (r *serpJobRepository) GetCachedJobByID(ctx context.Context, jobID, userID int32) (*domain.SerpJobCache, error) {
	row, err := r.queries.GetCachedJobByID(ctx, db.GetCachedJobByIDParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	c := mapSerpJob(row)
	return &c, nil
}

func (r *serpJobRepository) GetLatestFetchTime(ctx context.Context, userID int32) (*time.Time, error) {
	ts, err := r.queries.GetLatestCacheFetchTime(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}
	return &ts.Time, nil
}

func (r *serpJobRepository) MarkSavedToTracker(ctx context.Context, jobID, userID int32) error {
	return r.queries.MarkJobSavedToTracker(ctx, db.MarkJobSavedToTrackerParams{
		ID:     jobID,
		UserID: userID,
	})
}

func (r *serpJobRepository) DeleteOldCache(ctx context.Context, userID int32) error {
	return r.queries.DeleteOldCacheForUser(ctx, userID)
}

// ─────────────────────────────────────────
// Mappers & Helpers
// ─────────────────────────────────────────

func mapSerpJob(r db.SerpJobCache) domain.SerpJobCache {
	return domain.SerpJobCache{
		ID:             r.ID,
		UserID:         r.UserID,
		ExternalID:     r.ExternalID,
		Title:          r.Title,
		CompanyName:    derefString(r.CompanyName),
		Location:       derefString(r.Location),
		Description:    derefString(r.Description),
		Link:           derefString(r.Link),
		Platform:       derefString(r.Platform),
		PostedAt:       derefString(r.PostedAt),
		Salary:         derefString(r.Salary),
		SavedToTracker: r.SavedToTracker,
		FetchedAt:      r.FetchedAt.Time,
	}
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
