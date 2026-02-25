package service

import (
	"aiki/internal/domain"
	"aiki/internal/repository"
	"context"
	"fmt"
	"log"
	"time"
)

//go:generate mockgen -source=notification_service.go -destination=mocks/mock_notification_service.go -package=mocks

type NotificationService interface {
	// User-facing
	GetNotifications(ctx context.Context, userID int32, limit, offset int32) (*domain.NotificationSummary, error)
	GetUnreadCount(ctx context.Context, userID int32) (int32, error)
	MarkRead(ctx context.Context, notificationID, userID int32) error
	MarkAllRead(ctx context.Context, userID int32) error
	Delete(ctx context.Context, notificationID, userID int32) error

	// Internal triggers (called by other services)
	NotifySessionCompleted(ctx context.Context, userID int32, focusSeconds int32)
	NotifyStreakMilestone(ctx context.Context, userID int32, streak int32)
	NotifyBadgeEarned(ctx context.Context, userID int32, badgeName string)

	// Scheduled jobs
	SendDailyReminders(ctx context.Context)
	SendStreakWarnings(ctx context.Context)
}

type notificationService struct {
	notifRepo repository.NotificationRepository
}

func NewNotificationService(notifRepo repository.NotificationRepository) NotificationService {
	return &notificationService{notifRepo: notifRepo}
}

// ─────────────────────────────────────────
// User-facing
// ─────────────────────────────────────────

func (s *notificationService) GetNotifications(ctx context.Context, userID int32, limit, offset int32) (*domain.NotificationSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	notifications, err := s.notifRepo.GetUserNotifications(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	unreadCount, err := s.notifRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	if notifications == nil {
		notifications = []domain.Notification{}
	}

	return &domain.NotificationSummary{
		Notifications: notifications,
		UnreadCount:   unreadCount,
	}, nil
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID int32) (int32, error) {
	return s.notifRepo.GetUnreadCount(ctx, userID)
}

func (s *notificationService) MarkRead(ctx context.Context, notificationID, userID int32) error {
	return s.notifRepo.MarkRead(ctx, notificationID, userID)
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID int32) error {
	return s.notifRepo.MarkAllRead(ctx, userID)
}

func (s *notificationService) Delete(ctx context.Context, notificationID, userID int32) error {
	return s.notifRepo.Delete(ctx, notificationID, userID)
}

// ─────────────────────────────────────────
// Internal Triggers
// ─────────────────────────────────────────

func (s *notificationService) NotifySessionCompleted(ctx context.Context, userID int32, focusSeconds int32) {
	minutes := focusSeconds / 60
	title := "Session Complete 🔥"
	message := fmt.Sprintf("You stayed focused for %d minutes. Nice work!", minutes)

	// Add motivational messages based on focus time
	switch {
	case minutes >= 60:
		message = fmt.Sprintf("You focused for %d minutes. That's elite dedication!", minutes)
	case minutes >= 45:
		message = fmt.Sprintf("You stayed locked in for %d minutes. Great momentum!", minutes)
	case minutes >= 25:
		message = fmt.Sprintf("Solid %d minute session. Keep building consistency!", minutes)
	}

	_, err := s.notifRepo.Create(ctx, userID, domain.NotificationTypeSessionCompleted, title, message)
	if err != nil {
		log.Printf("failed to create session completed notification for user %d: %v", userID, err)
	}
}

func (s *notificationService) NotifyStreakMilestone(ctx context.Context, userID int32, streak int32) {
	// Only notify on specific milestones
	milestones := map[int32][2]string{
		3:   {"3-Day Streak! 🎯", "You're building a habit. 3 days strong!"},
		7:   {"7-Day Streak! 💪", "One full week of consistency. You're on fire!"},
		14:  {"2-Week Streak! ⚡", "Two weeks straight. Your focus is unstoppable!"},
		30:  {"30-Day Streak! 🏆", "30 days straight. That's elite focus!"},
		60:  {"60-Day Streak! 👑", "Two months of daily sessions. Absolutely legendary!"},
		100: {"100-Day Streak! 🚀", "100 days. You've built something truly special!"},
	}

	content, isMilestone := milestones[streak]
	if !isMilestone {
		return
	}

	_, err := s.notifRepo.Create(ctx, userID, domain.NotificationTypeStreakMilestone, content[0], content[1])
	if err != nil {
		log.Printf("failed to create streak milestone notification for user %d: %v", userID, err)
	}
}

func (s *notificationService) NotifyBadgeEarned(ctx context.Context, userID int32, badgeName string) {
	title := "Badge Unlocked! 🏅"
	message := fmt.Sprintf("You just earned the \"%s\" badge. Keep it up!", badgeName)

	_, err := s.notifRepo.Create(ctx, userID, domain.NotificationTypeBadgeEarned, title, message)
	if err != nil {
		log.Printf("failed to create badge earned notification for user %d: %v", userID, err)
	}
}

// ─────────────────────────────────────────
// Scheduled Jobs
// ─────────────────────────────────────────

// SendDailyReminders sends a reminder to all users who haven't completed
// a session today. Call this from a cron job (e.g. every day at 6PM).
func (s *notificationService) SendDailyReminders(ctx context.Context) {
	userIDs, err := s.notifRepo.GetUsersWithNoSessionToday(ctx)
	if err != nil {
		log.Printf("failed to get users with no session today: %v", err)
		return
	}

	title := "Time to Lock In 🔒"
	message := s.getDailyReminderMessage()

	for _, userID := range userIDs {
		_, err := s.notifRepo.Create(ctx, userID, domain.NotificationTypeDailyReminder, title, message)
		if err != nil {
			log.Printf("failed to create daily reminder for user %d: %v", userID, err)
		}
	}

	log.Printf("daily reminders sent to %d users", len(userIDs))
}

// SendStreakWarnings sends a warning to users with an active streak who
// haven't sessioned today. Call this from a cron job (e.g. every day at 8PM).
func (s *notificationService) SendStreakWarnings(ctx context.Context) {
	userIDs, err := s.notifRepo.GetUsersWithNoSessionToday(ctx)
	if err != nil {
		log.Printf("failed to get users for streak warning: %v", err)
		return
	}

	title := "Streak at Risk ⚠️"
	message := "Don't let your streak slip away! Complete a session before midnight to keep it alive."

	for _, userID := range userIDs {
		_, err := s.notifRepo.Create(ctx, userID, domain.NotificationTypeStreakWarning, title, message)
		if err != nil {
			log.Printf("failed to create streak warning for user %d: %v", userID, err)
		}
	}

	log.Printf("streak warnings sent to %d users", len(userIDs))
}

// getDailyReminderMessage rotates motivational messages
func (s *notificationService) getDailyReminderMessage() string {
	messages := []string{
		"You haven't locked in today. Start a session and keep the momentum going!",
		"Consistency is built one session at a time. Lock in today!",
		"Your future self will thank you. Start a focus session now!",
		"Even 25 minutes of focus today makes a difference. Let's go!",
		"Don't break the chain. Lock in and keep your streak alive!",
	}
	// Rotate based on day of year
	day := time.Now().YearDay()
	return messages[day%len(messages)]
}
