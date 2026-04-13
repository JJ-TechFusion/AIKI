package repository

import (
	"aiki/internal/database/db"
	"aiki/internal/domain"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var logger = log.Logger{}

type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) (int32, error)
	Update(ctx context.Context, jobId int32, job *domain.Job) error
	DeleteJob(ctx context.Context, jobId int32) error
	GetJobByID(ctx context.Context, jobId int32) (*domain.Job, error)
	GetAllJobs(ctx context.Context, userId int32) ([]db.Job, error)
}

type jobRepository struct {
	db *db.Queries
}

func NewJobRepository(dbPool *pgxpool.Pool) JobRepository {
	return &jobRepository{db: db.New(dbPool)}
}

func (jr *jobRepository) Create(ctx context.Context, job *domain.Job) (int32, error) {
	var dateApplied pgtype.Timestamp
	if job.DateApplied != "" {
		t, err := time.Parse("2006-01-02", job.DateApplied)
		if err != nil {
			return 0, domain.ErrInvalidDateFormat
		}
		dateApplied = PgTimeHelper(t)
	}

	newJob := db.CreateJobParams{
		UserID:      job.UserId,
		Title:       job.Title,
		CompanyName: &job.CompanyName,
		Notes:       &job.Notes,
		Link:        nullableString(job.Link),
		Location:    &job.Location,
		Platform:    &job.Platform,
		DateApplied: dateApplied,
		Status:      job.Status,
	}
	createdJob, err := jr.db.CreateJob(ctx, newJob)
	if err != nil {
		fmt.Println("failed to create job, error:", err)
		return 0, domain.ErrFailedToCreateJob
	}
	return createdJob.ID, nil
}

func (jr *jobRepository) Update(ctx context.Context, jobId int32, job *domain.Job) error {
	var dateApplied pgtype.Timestamp
	if job.DateApplied != "" {
		t, err := time.Parse("2006-01-02", job.DateApplied)
		if err != nil {
			return domain.ErrInvalidDateFormat
		}
		dateApplied = PgTimeHelper(t)
	}
	err := jr.db.UpdateJobByID(ctx, db.UpdateJobByIDParams{
		ID:          jobId,
		Title:       &job.Title,
		CompanyName: &job.CompanyName,
		Notes:       &job.Notes,
		Link:        nullableString(job.Link),
		Location:    &job.Location,
		Platform:    &job.Platform,
		DateApplied: dateApplied,
		Status:      &job.Status,
	})
	if err != nil {
		fmt.Println("failed to update job with id:", jobId, err)
		return domain.ErrFailedToUpdateJob
	}
	return nil
}

func (jr *jobRepository) DeleteJob(ctx context.Context, jobId int32) error {
	err := jr.db.DeleteJobByID(ctx, jobId)
	if err != nil {
		fmt.Println("failed to delete job with id:", jobId, err)
		return domain.ErrFailedToUpdateJob
	}
	return nil
}

func (jr *jobRepository) GetJobByID(ctx context.Context, jobId int32) (*domain.Job, error) {
	job, err := jr.db.GetJobByID(ctx, jobId)
	if err != nil {
		return &domain.Job{}, domain.ErrInvalidJobID
	}
	out := &domain.Job{
		ID:          jobId,
		UserId:      job.UserID,
		Title:       job.Title,
		CompanyName: derefString(job.CompanyName),
		Notes:       derefString(job.Notes),
		Location:    derefString(job.Location),
		Status:      job.Status,
		Link:        derefString(job.Link),
		Platform:    derefString(job.Platform),
		CreatedAt:   job.CreatedAt.Time,
	}
	if job.DateApplied.Valid {
		out.DateApplied = job.DateApplied.Time.Format("2006-01-02")
	}
	return out, nil
}

func (jr *jobRepository) GetAllJobs(ctx context.Context, userId int32) ([]db.Job, error) {
	return jr.db.GetJobs(ctx, userId)
}

func PgTimeHelper(data time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: data, Valid: true}
}
