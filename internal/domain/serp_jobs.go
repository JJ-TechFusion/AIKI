package domain

import "time"

// SerpJob represents a job fetched from SerpApi
type SerpJob struct {
	ExternalID  string    `json:"external_id"`
	Title       string    `json:"title"`
	CompanyName string    `json:"company_name"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	Platform    string    `json:"platform"`
	PostedAt    string    `json:"posted_at"`
	Salary      string    `json:"salary,omitempty"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// SerpJobCache is a fetched job stored in the DB
type SerpJobCache struct {
	ID             int32     `json:"id"`
	UserID         int32     `json:"user_id"`
	ExternalID     string    `json:"external_id"`
	Title          string    `json:"title"`
	CompanyName    string    `json:"company_name"`
	Location       string    `json:"location"`
	Description    string    `json:"description"`
	Link           string    `json:"link"`
	Platform       string    `json:"platform"`
	PostedAt       string    `json:"posted_at"`
	Salary         string    `json:"salary"`
	SavedToTracker bool      `json:"saved_to_tracker"`
	TrackerJobID   *int32    `json:"tracker_job_id,omitempty"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// JobSearchResult is the response returned to the client
type JobSearchResult struct {
	Jobs       []SerpJobCache `json:"jobs"`
	TotalCount int            `json:"total_count"`
	FromCache  bool           `json:"from_cache"`
	FetchedAt  time.Time      `json:"fetched_at"`
}

// SaveJobRequest is used to save a fetched job to the tracker
type SaveJobRequest struct {
	CacheID int32 `json:"cache_id" validate:"required"`
}
