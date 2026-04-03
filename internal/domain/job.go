package domain

import (
	"time"
)

const (
	JobStatusSaved     = "saved"
	JobStatusApplied   = "applied"
	JobStatusInterview = "interview"
	JobStatusOffer     = "offer"
	JobStatusRejected  = "rejected"
)

type Job struct {
	ID          int32     `json:"id"`
	UserId      int32     `json:"user_id"`
	Title       string    `json:"title"`
	CompanyName string    `json:"company_name"`
	Location    string    `json:"location"`
	Platform    string    `json:"platform"` // could be linkedin, website or indeed or others
	Link        string    `json:"link"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
	DateApplied string    `json:"date_applied"`
	CreatedAt   time.Time `json:"created_at"`
}

type JobRequest struct {
	Title       string `json:"title"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	Platform    string `json:"platform"` // could be linkedin, website or indeed or others
	Link        string `json:"link"`
	Notes       string `json:"notes"`
	Status      string `json:"status"`
	DateApplied string `json:"date_applied"` // DD-MM-YYYY
}

type DirectApplyRequest struct {
	Notes string `json:"notes"`
}

type DirectApplyResult struct {
	Job      Job    `json:"job"`
	ApplyURL string `json:"apply_url"`
}

func (j JobRequest) ToDomain(userId int32) Job {
	return Job{
		UserId:      userId,
		Title:       j.Title,
		CompanyName: j.CompanyName,
		Location:    j.Location,
		Platform:    j.Platform,
		Link:        j.Link,
		Status:      j.Status,
		Notes:       j.Notes,
		DateApplied: j.DateApplied,
	}
}
