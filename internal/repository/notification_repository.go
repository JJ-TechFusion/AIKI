package repository

import (
	"aiki/internal/database/db"
	"aiki/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=notification_repository.go -destination=mocks/mock_notification_repository.go -package=mocks

type NotificationRepository interface {
	Create(ctx context.Context, userID int32, notifType domain.NotificationType, title, message string) (*domain.Notification, error)
	GetUserNotifications(ctx context.Context, userID int32, limit, offset int32) ([]domain.Notification, error)
	GetUnreadCount(ctx context.Context, userID int32) (int32, error)
	MarkRead(ctx context.Context, notificationID, userID int32) error
	MarkAllRead(ctx context.Context, userID int32) error
	Delete(ctx context.Context, notificationID, userID int32) error
	GetPreferences(ctx context.Context, userID int32) (*domain.NotificationPreferences, error)
	UpsertPreferences(ctx context.Context, prefs domain.NotificationPreferences) (*domain.NotificationPreferences, error)
	GetUsersWithNoSessionToday(ctx context.Context) ([]int32, error)
	GetDailyReminderRecipients(ctx context.Context) ([]int32, error)
	GetStreakWarningRecipients(ctx context.Context) ([]int32, error)
}

type notificationRepository struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func NewNotificationRepository(dbPool *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{db: dbPool, queries: db.New(dbPool)}
}

func (r *notificationRepository) Create(ctx context.Context, userID int32, notifType domain.NotificationType, title, message string) (*domain.Notification, error) {
	row, err := r.queries.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:  userID,
		Type:    string(notifType),
		Title:   title,
		Message: message,
	})
	if err != nil {
		return nil, err
	}
	return mapNotification(row), nil
}

func (r *notificationRepository) GetUserNotifications(ctx context.Context, userID int32, limit, offset int32) ([]domain.Notification, error) {
	rows, err := r.queries.GetUserNotifications(ctx, db.GetUserNotificationsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	notifications := make([]domain.Notification, len(rows))
	for i, row := range rows {
		notifications[i] = *mapNotification(row)
	}
	return notifications, nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID int32) (int32, error) {
	count, err := r.queries.GetUnreadCount(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, notificationID, userID int32) error {
	return r.queries.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		ID:     notificationID,
		UserID: userID,
	})
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID int32) error {
	return r.queries.MarkAllNotificationsRead(ctx, userID)
}

func (r *notificationRepository) Delete(ctx context.Context, notificationID, userID int32) error {
	return r.queries.DeleteNotification(ctx, db.DeleteNotificationParams{
		ID:     notificationID,
		UserID: userID,
	})
}

func (r *notificationRepository) GetUsersWithNoSessionToday(ctx context.Context) ([]int32, error) {
	return r.queries.GetUsersWithNoSessionToday(ctx)
}

func (r *notificationRepository) GetPreferences(ctx context.Context, userID int32) (*domain.NotificationPreferences, error) {
	const ensureQuery = `
		INSERT INTO notification_preferences (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`
	if _, err := r.db.Exec(ctx, ensureQuery, userID); err != nil {
		return nil, err
	}

	const query = `
		SELECT
			user_id,
			in_app_enabled,
			push_enabled,
			email_enabled,
			session_completed,
			daily_reminder,
			motivational_nudges,
			streak_milestone,
			streak_warning,
			badge_earned,
			follow_up_reminder,
			application_check_in,
			interview_reminder,
			updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	return scanNotificationPreferences(r.db.QueryRow(ctx, query, userID))
}

func (r *notificationRepository) UpsertPreferences(ctx context.Context, prefs domain.NotificationPreferences) (*domain.NotificationPreferences, error) {
	const query = `
		INSERT INTO notification_preferences (
			user_id,
			in_app_enabled,
			push_enabled,
			email_enabled,
			session_completed,
			daily_reminder,
			motivational_nudges,
			streak_milestone,
			streak_warning,
			badge_earned,
			follow_up_reminder,
			application_check_in,
			interview_reminder
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (user_id) DO UPDATE SET
			in_app_enabled = EXCLUDED.in_app_enabled,
			push_enabled = EXCLUDED.push_enabled,
			email_enabled = EXCLUDED.email_enabled,
			session_completed = EXCLUDED.session_completed,
			daily_reminder = EXCLUDED.daily_reminder,
			motivational_nudges = EXCLUDED.motivational_nudges,
			streak_milestone = EXCLUDED.streak_milestone,
			streak_warning = EXCLUDED.streak_warning,
			badge_earned = EXCLUDED.badge_earned,
			follow_up_reminder = EXCLUDED.follow_up_reminder,
			application_check_in = EXCLUDED.application_check_in,
			interview_reminder = EXCLUDED.interview_reminder,
			updated_at = NOW()
		RETURNING
			user_id,
			in_app_enabled,
			push_enabled,
			email_enabled,
			session_completed,
			daily_reminder,
			motivational_nudges,
			streak_milestone,
			streak_warning,
			badge_earned,
			follow_up_reminder,
			application_check_in,
			interview_reminder,
			updated_at
	`

	return scanNotificationPreferences(
		r.db.QueryRow(
			ctx,
			query,
			prefs.UserID,
			prefs.InAppEnabled,
			prefs.PushEnabled,
			prefs.EmailEnabled,
			prefs.SessionCompleted,
			prefs.DailyReminder,
			prefs.MotivationalNudges,
			prefs.StreakMilestone,
			prefs.StreakWarning,
			prefs.BadgeEarned,
			prefs.FollowUpReminder,
			prefs.ApplicationCheckIn,
			prefs.InterviewReminder,
		),
	)
}

func (r *notificationRepository) GetDailyReminderRecipients(ctx context.Context) ([]int32, error) {
	const query = `
		SELECT u.id
		FROM users u
		LEFT JOIN notification_preferences np ON np.user_id = u.id
		WHERE u.is_active = TRUE
		  AND NOT EXISTS (
			SELECT 1
			FROM focus_sessions fs
			WHERE fs.user_id = u.id
			  AND fs.status = 'completed'
			  AND DATE(fs.ended_at) = CURRENT_DATE
		  )
		  AND COALESCE(np.in_app_enabled, TRUE) = TRUE
		  AND COALESCE(np.daily_reminder, TRUE) = TRUE
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int32
	for rows.Next() {
		var userID int32
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

func (r *notificationRepository) GetStreakWarningRecipients(ctx context.Context) ([]int32, error) {
	const query = `
		SELECT u.id
		FROM users u
		INNER JOIN streaks s ON s.user_id = u.id AND s.current_streak > 0
		LEFT JOIN notification_preferences np ON np.user_id = u.id
		WHERE u.is_active = TRUE
		  AND NOT EXISTS (
			SELECT 1
			FROM focus_sessions fs
			WHERE fs.user_id = u.id
			  AND fs.status = 'completed'
			  AND DATE(fs.ended_at) = CURRENT_DATE
		  )
		  AND COALESCE(np.in_app_enabled, TRUE) = TRUE
		  AND COALESCE(np.streak_warning, TRUE) = TRUE
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int32
	for rows.Next() {
		var userID int32
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// ─────────────────────────────────────────
// Mapper
// ─────────────────────────────────────────

func mapNotification(n db.Notification) *domain.Notification {
	return &domain.Notification{
		ID:        n.ID,
		UserID:    n.UserID,
		Type:      domain.NotificationType(n.Type),
		Title:     n.Title,
		Message:   n.Message,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.Time,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotificationPreferences(scanner rowScanner) (*domain.NotificationPreferences, error) {
	var prefs domain.NotificationPreferences
	err := scanner.Scan(
		&prefs.UserID,
		&prefs.InAppEnabled,
		&prefs.PushEnabled,
		&prefs.EmailEnabled,
		&prefs.SessionCompleted,
		&prefs.DailyReminder,
		&prefs.MotivationalNudges,
		&prefs.StreakMilestone,
		&prefs.StreakWarning,
		&prefs.BadgeEarned,
		&prefs.FollowUpReminder,
		&prefs.ApplicationCheckIn,
		&prefs.InterviewReminder,
		&prefs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &prefs, nil
}

// compile-time check
var _ NotificationRepository = (*notificationRepository)(nil)
