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
	GetUsersWithNoSessionToday(ctx context.Context) ([]int32, error)
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

// compile-time check
var _ NotificationRepository = (*notificationRepository)(nil)
