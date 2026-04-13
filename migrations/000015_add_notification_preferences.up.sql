CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id               INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    in_app_enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    push_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    email_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    session_completed     BOOLEAN NOT NULL DEFAULT TRUE,
    daily_reminder        BOOLEAN NOT NULL DEFAULT TRUE,
    motivational_nudges   BOOLEAN NOT NULL DEFAULT TRUE,
    streak_milestone      BOOLEAN NOT NULL DEFAULT TRUE,
    streak_warning        BOOLEAN NOT NULL DEFAULT TRUE,
    badge_earned          BOOLEAN NOT NULL DEFAULT TRUE,
    follow_up_reminder    BOOLEAN NOT NULL DEFAULT TRUE,
    application_check_in  BOOLEAN NOT NULL DEFAULT TRUE,
    interview_reminder    BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at            TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_notification_preferences_updated_at
BEFORE UPDATE ON notification_preferences
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
