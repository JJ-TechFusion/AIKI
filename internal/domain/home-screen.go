package domain

import "time"

// ─────────────────────────────────────────
// Focus Session
// ─────────────────────────────────────────

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusPaused    SessionStatus = "paused"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusAbandoned SessionStatus = "abandoned"
)

type FocusSession struct {
	ID              int32         `json:"id"`
	UserID          int32         `json:"user_id"`
	DurationSeconds int32         `json:"duration_seconds"` // planned
	ElapsedSeconds  int32         `json:"elapsed_seconds"`  // completed so far
	Status          SessionStatus `json:"status"`
	StartedAt       time.Time     `json:"started_at"`
	EndedAt         *time.Time    `json:"ended_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type StartSessionRequest struct {
	DurationSeconds int32 `json:"duration_seconds" validate:"required,min=60,max=86400"`
}

type UpdateSessionRequest struct {
	ElapsedSeconds int32         `json:"elapsed_seconds" validate:"required,min=0"`
	Status         SessionStatus `json:"status" validate:"required,oneof=active paused completed abandoned"`
}

// ─────────────────────────────────────────
// Streak
// ─────────────────────────────────────────

type Streak struct {
	UserID          int32      `json:"user_id"`
	CurrentStreak   int32      `json:"current_streak"`
	LongestStreak   int32      `json:"longest_streak"`
	LastSessionDate *time.Time `json:"last_session_date,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─────────────────────────────────────────
// Badges
// ─────────────────────────────────────────

type BadgeDefinition struct {
	ID            int32     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	IconKey       string    `json:"icon_key"`
	CriteriaType  string    `json:"criteria_type"`
	CriteriaValue int32     `json:"criteria_value"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserBadge struct {
	BadgeID     int32     `json:"badge_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IconKey     string    `json:"icon_key"`
	EarnedAt    time.Time `json:"earned_at"`
}

// ─────────────────────────────────────────
// Progress / Stats
// ─────────────────────────────────────────

type DailyProgress struct {
	UserID            int32     `json:"user_id"`
	Date              time.Time `json:"date"`
	TotalFocusSeconds int32     `json:"total_focus_seconds"`
	SessionsCompleted int32     `json:"sessions_completed"`
}

type ProgressSummary struct {
	TotalFocusSeconds int64                `json:"total_focus_seconds"`
	TotalFocusHours   float64              `json:"total_focus_hours"`
	SessionsCompleted int64                `json:"sessions_completed"`
	DaysActive        int64                `json:"days_active"`
	Period            string               `json:"period"` // weekly | monthly | yearly
	ChartPoints       []ProgressChartPoint `json:"chart_points"`
}

type ProgressChartPoint struct {
	Label             string  `json:"label"`
	BucketKey         string  `json:"bucket_key"`
	TotalFocusSeconds int64   `json:"total_focus_seconds"`
	TotalFocusHours   float64 `json:"total_focus_hours"`
	SessionsCompleted int64   `json:"sessions_completed"`
}

// HomeScreenData is the aggregated payload for the home screen
type HomeScreenData struct {
	Streak         *Streak         `json:"streak"`
	ActiveSession  *FocusSession   `json:"active_session,omitempty"`
	WeeklyProgress ProgressSummary `json:"weekly_progress"`
	RecentBadges   []UserBadge     `json:"recent_badges"`
	TotalBadges    int32           `json:"total_badges"`
}
