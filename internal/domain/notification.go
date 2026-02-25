package domain

import "time"

type NotificationType string

const (
	NotificationTypeSessionCompleted NotificationType = "session_completed"
	NotificationTypeStreakMilestone  NotificationType = "streak_milestone"
	NotificationTypeBadgeEarned      NotificationType = "badge_earned"
	NotificationTypeDailyReminder    NotificationType = "daily_reminder"
	NotificationTypeStreakWarning    NotificationType = "streak_warning"
)

type Notification struct {
	ID        int32            `json:"id"`
	UserID    int32            `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	IsRead    bool             `json:"is_read"`
	CreatedAt time.Time        `json:"created_at"`
}

type NotificationSummary struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int32          `json:"unread_count"`
}

// Notification messages per trigger
var notificationMessages = map[string][2]string{
	// type: [title, message]
	"session_completed": {"Session Complete 🔥", "You stayed focused! Keep the momentum going."},
	"streak_3":          {"3-Day Streak! 🎯", "You're building a habit. 3 days strong!"},
	"streak_7":          {"7-Day Streak! 💪", "One full week of consistency. You're on fire!"},
	"streak_30":         {"30-Day Streak! 🏆", "30 days straight. That's elite focus!"},
	"badge_earned":      {"Badge Unlocked! 🏅", "You earned a new badge. Check it out!"},
	"daily_reminder":    {"Time to Lock In 🔒", "You haven't sessioned today. Don't break your streak!"},
	"streak_warning":    {"Streak at Risk ⚠️", "Complete a session today to keep your streak alive!"},
}

func GetNotificationContent(notifType string, extra ...string) (title, message string) {
	content, ok := notificationMessages[notifType]
	if !ok {
		return "Aiki", "You have a new notification."
	}
	title = content[0]
	message = content[1]

	// Override message with custom content if provided
	if len(extra) >= 1 && extra[0] != "" {
		title = extra[0]
	}
	if len(extra) >= 2 && extra[1] != "" {
		message = extra[1]
	}
	return title, message
}
