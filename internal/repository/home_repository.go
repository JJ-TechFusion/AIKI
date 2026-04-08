package repository

import (
	"aiki/internal/database/db"
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=home_repository.go -destination=mocks/mock_home_repository.go -package=mocks

type HomeRepository interface {
	// Focus Sessions
	CreateSession(ctx context.Context, userID int32, durationSeconds int32) (*domain.FocusSession, error)
	GetSessionByID(ctx context.Context, sessionID int32) (*domain.FocusSession, error)
	GetActiveSession(ctx context.Context, userID int32) (*domain.FocusSession, error)
	UpdateSession(ctx context.Context, sessionID int32, elapsedSeconds int32, status domain.SessionStatus, endedAt *time.Time) (*domain.FocusSession, error)
	GetUserSessionHistory(ctx context.Context, userID int32, limit, offset int32) ([]domain.FocusSession, error)

	// Streaks
	GetStreak(ctx context.Context, userID int32) (*domain.Streak, error)
	UpsertStreak(ctx context.Context, userID int32, currentStreak, longestStreak int32, lastDate *time.Time) (*domain.Streak, error)

	// Badges
	GetAllBadgeDefinitions(ctx context.Context) ([]domain.BadgeDefinition, error)
	GetUserBadges(ctx context.Context, userID int32) ([]domain.UserBadge, error)
	AwardBadge(ctx context.Context, userID, badgeID int32) error
	GetUserBadgeCount(ctx context.Context, userID int32) (int32, error)

	// Progress
	UpsertDailyProgress(ctx context.Context, userID int32, date time.Time, focusSeconds, sessions int32) error
	GetProgressSummary(ctx context.Context, userID int32, from, to time.Time) (*domain.ProgressSummary, error)
	GetDailyProgressRange(ctx context.Context, userID int32, from, to time.Time) ([]domain.DailyProgress, error)
}

type homeRepository struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func NewHomeRepository(dbPool *pgxpool.Pool) HomeRepository {
	return &homeRepository{db: dbPool, queries: db.New(dbPool)}
}

// ─────────────────────────────────────────
// Focus Sessions
// ─────────────────────────────────────────

func (r *homeRepository) CreateSession(ctx context.Context, userID int32, durationSeconds int32) (*domain.FocusSession, error) {
	session, err := r.queries.CreateFocusSession(ctx, db.CreateFocusSessionParams{
		UserID:          userID,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return nil, err
	}
	return mapSession(session), nil
}

func (r *homeRepository) GetSessionByID(ctx context.Context, sessionID int32) (*domain.FocusSession, error) {
	session, err := r.queries.GetFocusSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, response.ErrSessionNotFound
		}
		return nil, err
	}
	return mapSession(session), nil
}

func (r *homeRepository) GetActiveSession(ctx context.Context, userID int32) (*domain.FocusSession, error) {
	session, err := r.queries.GetActiveSession(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return mapSession(session), nil
}

func (r *homeRepository) UpdateSession(ctx context.Context, sessionID int32, elapsedSeconds int32, status domain.SessionStatus, endedAt *time.Time) (*domain.FocusSession, error) {
	var pgEndedAt pgtype.Timestamp
	if endedAt != nil {
		pgEndedAt = pgtype.Timestamp{Time: *endedAt, Valid: true}
	}

	session, err := r.queries.UpdateFocusSession(ctx, db.UpdateFocusSessionParams{
		ID:             sessionID,
		ElapsedSeconds: elapsedSeconds,
		Status:         string(status),
		EndedAt:        pgEndedAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, response.ErrSessionNotFound
		}
		return nil, err
	}
	return mapSession(session), nil
}

func (r *homeRepository) GetUserSessionHistory(ctx context.Context, userID int32, limit, offset int32) ([]domain.FocusSession, error) {
	rows, err := r.queries.GetUserSessionHistory(ctx, db.GetUserSessionHistoryParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]domain.FocusSession, len(rows))
	for i, row := range rows {
		sessions[i] = *mapSession(row)
	}
	return sessions, nil
}

// ─────────────────────────────────────────
// Streaks
// ─────────────────────────────────────────

func (r *homeRepository) GetStreak(ctx context.Context, userID int32) (*domain.Streak, error) {
	streak, err := r.queries.GetStreak(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.Streak{UserID: userID}, nil
		}
		return nil, err
	}
	return mapStreak(streak), nil
}

func (r *homeRepository) UpsertStreak(ctx context.Context, userID int32, currentStreak, longestStreak int32, lastDate *time.Time) (*domain.Streak, error) {
	var pgDate pgtype.Date
	if lastDate != nil {
		pgDate = pgtype.Date{Time: *lastDate, Valid: true}
	}

	streak, err := r.queries.UpsertStreak(ctx, db.UpsertStreakParams{
		UserID:          userID,
		CurrentStreak:   currentStreak,
		LongestStreak:   longestStreak,
		LastSessionDate: pgDate,
	})
	if err != nil {
		return nil, err
	}
	return mapStreak(streak), nil
}

// ─────────────────────────────────────────
// Badges
// ─────────────────────────────────────────

func (r *homeRepository) GetAllBadgeDefinitions(ctx context.Context) ([]domain.BadgeDefinition, error) {
	rows, err := r.queries.GetAllBadgeDefinitions(ctx)
	if err != nil {
		return nil, err
	}

	badges := make([]domain.BadgeDefinition, len(rows))
	for i, row := range rows {
		badges[i] = domain.BadgeDefinition{
			ID:            row.ID,
			Name:          row.Name,
			Description:   derefString(row.Description),
			IconKey:       derefString(row.IconKey),
			CriteriaType:  row.CriteriaType,
			CriteriaValue: row.CriteriaValue,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return badges, nil
}

func (r *homeRepository) GetUserBadges(ctx context.Context, userID int32) ([]domain.UserBadge, error) {
	rows, err := r.queries.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	badges := make([]domain.UserBadge, len(rows))
	for i, row := range rows {
		badges[i] = domain.UserBadge{
			BadgeID:     row.ID,
			Name:        row.Name,
			Description: derefString(row.Description),
			IconKey:     derefString(row.IconKey),
			EarnedAt:    row.EarnedAt.Time,
		}
	}
	return badges, nil
}

func (r *homeRepository) AwardBadge(ctx context.Context, userID, badgeID int32) error {
	return r.queries.AwardBadge(ctx, db.AwardBadgeParams{
		UserID:  userID,
		BadgeID: badgeID,
	})
}

func (r *homeRepository) GetUserBadgeCount(ctx context.Context, userID int32) (int32, error) {
	count, err := r.queries.GetUserBadgeCount(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// ─────────────────────────────────────────
// Progress
// ─────────────────────────────────────────

func (r *homeRepository) UpsertDailyProgress(ctx context.Context, userID int32, date time.Time, focusSeconds, sessions int32) error {
	return r.queries.UpsertDailyProgress(ctx, db.UpsertDailyProgressParams{
		UserID:            userID,
		Date:              pgtype.Date{Time: date, Valid: true},
		TotalFocusSeconds: focusSeconds,
		SessionsCompleted: sessions,
	})
}

func (r *homeRepository) GetProgressSummary(ctx context.Context, userID int32, from, to time.Time) (*domain.ProgressSummary, error) {
	row, err := r.queries.GetProgressSummary(ctx, db.GetProgressSummaryParams{
		UserID: userID,
		Date:   pgtype.Date{Time: from, Valid: true},
		Date_2: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &domain.ProgressSummary{
		TotalFocusSeconds: row.TotalFocusSeconds,
		TotalFocusHours:   float64(row.TotalFocusSeconds) / 3600.0,
		SessionsCompleted: row.SessionsCompleted,
		DaysActive:        row.DaysActive,
	}, nil
}

func (r *homeRepository) GetDailyProgressRange(ctx context.Context, userID int32, from, to time.Time) ([]domain.DailyProgress, error) {
	const query = `
		SELECT date, total_focus_seconds, sessions_completed
		FROM daily_progress
		WHERE user_id = $1
		  AND date BETWEEN $2 AND $3
		ORDER BY date ASC
	`

	rows, err := r.db.Query(ctx, query, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyProgress []domain.DailyProgress
	for rows.Next() {
		var date pgtype.Date
		var focusSeconds int32
		var sessionsCompleted int32

		if err := rows.Scan(&date, &focusSeconds, &sessionsCompleted); err != nil {
			return nil, err
		}

		dailyProgress = append(dailyProgress, domain.DailyProgress{
			UserID:            userID,
			Date:              date.Time,
			TotalFocusSeconds: focusSeconds,
			SessionsCompleted: sessionsCompleted,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dailyProgress, nil
}

// ─────────────────────────────────────────
// Mappers
// ─────────────────────────────────────────

func mapSession(s db.FocusSession) *domain.FocusSession {
	var endedAt *time.Time
	if s.EndedAt.Valid {
		endedAt = &s.EndedAt.Time
	}
	return &domain.FocusSession{
		ID:              s.ID,
		UserID:          s.UserID,
		DurationSeconds: s.DurationSeconds,
		ElapsedSeconds:  s.ElapsedSeconds,
		Status:          domain.SessionStatus(s.Status),
		StartedAt:       s.StartedAt.Time,
		EndedAt:         endedAt,
		CreatedAt:       s.CreatedAt.Time,
		UpdatedAt:       s.UpdatedAt.Time,
	}
}

func mapStreak(s db.Streak) *domain.Streak {
	var lastDate *time.Time
	if s.LastSessionDate.Valid {
		t := s.LastSessionDate.Time
		lastDate = &t
	}
	return &domain.Streak{
		UserID:          s.UserID,
		CurrentStreak:   s.CurrentStreak,
		LongestStreak:   s.LongestStreak,
		LastSessionDate: lastDate,
		UpdatedAt:       s.UpdatedAt.Time,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
