package domain

import "time"

type NotificationPreferences struct {
	UserID             int32     `json:"user_id,omitempty"`
	InAppEnabled       bool      `json:"in_app_enabled"`
	PushEnabled        bool      `json:"push_enabled"`
	EmailEnabled       bool      `json:"email_enabled"`
	SessionCompleted   bool      `json:"session_completed"`
	DailyReminder      bool      `json:"daily_reminder"`
	MotivationalNudges bool      `json:"motivational_nudges"`
	StreakMilestone    bool      `json:"streak_milestone"`
	StreakWarning      bool      `json:"streak_warning"`
	BadgeEarned        bool      `json:"badge_earned"`
	FollowUpReminder   bool      `json:"follow_up_reminder"`
	ApplicationCheckIn bool      `json:"application_check_in"`
	InterviewReminder  bool      `json:"interview_reminder"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateNotificationPreferencesRequest struct {
	InAppEnabled       bool `json:"in_app_enabled"`
	PushEnabled        bool `json:"push_enabled"`
	EmailEnabled       bool `json:"email_enabled"`
	SessionCompleted   bool `json:"session_completed"`
	DailyReminder      bool `json:"daily_reminder"`
	MotivationalNudges bool `json:"motivational_nudges"`
	StreakMilestone    bool `json:"streak_milestone"`
	StreakWarning      bool `json:"streak_warning"`
	BadgeEarned        bool `json:"badge_earned"`
	FollowUpReminder   bool `json:"follow_up_reminder"`
	ApplicationCheckIn bool `json:"application_check_in"`
	InterviewReminder  bool `json:"interview_reminder"`
}

func DefaultNotificationPreferences(userID int32) NotificationPreferences {
	return NotificationPreferences{
		UserID:             userID,
		InAppEnabled:       true,
		PushEnabled:        false,
		EmailEnabled:       false,
		SessionCompleted:   true,
		DailyReminder:      true,
		MotivationalNudges: true,
		StreakMilestone:    true,
		StreakWarning:      true,
		BadgeEarned:        true,
		FollowUpReminder:   true,
		ApplicationCheckIn: true,
		InterviewReminder:  true,
	}
}

func (p NotificationPreferences) AllowsInApp(notifType NotificationType) bool {
	if !p.InAppEnabled {
		return false
	}

	switch notifType {
	case NotificationTypeSessionCompleted:
		return p.SessionCompleted
	case NotificationTypeStreakMilestone:
		return p.StreakMilestone
	case NotificationTypeBadgeEarned:
		return p.BadgeEarned
	case NotificationTypeDailyReminder:
		return p.DailyReminder
	case NotificationTypeStreakWarning:
		return p.StreakWarning
	default:
		return true
	}
}
