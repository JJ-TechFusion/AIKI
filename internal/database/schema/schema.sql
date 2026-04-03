-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone_number VARCHAR(20),
    password_hash VARCHAR(255), -- Made nullable
    linkedin_id TEXT, -- Added new column
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = TRUE;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- User profile table
CREATE TABLE IF NOT EXISTS user_profile (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE REFERENCES users(id),
    cv BYTEA,
    full_name VARCHAR(200),
    current_job VARCHAR(255),
    experience_level VARCHAR(100),
    goals TEXT[] DEFAULT '{}',
    job_search_location TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    company_name VARCHAR(150),
    notes TEXT,
    link TEXT,
    location VARCHAR(255),
    platform VARCHAR(100),
    date_applied TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'applied',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Home Screen Features
-- Focus Sessions, Streaks, Badges, Progress Stats
-- ============================================================

-- Focus sessions table
CREATE TABLE IF NOT EXISTS focus_sessions (
    id               SERIAL PRIMARY KEY,
    user_id          INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    duration_seconds INT NOT NULL DEFAULT 0,
    elapsed_seconds  INT NOT NULL DEFAULT 0,
    status           VARCHAR(20) NOT NULL DEFAULT 'active', -- active | paused | completed | abandoned
    started_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMP,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_focus_sessions_user_id    ON focus_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_focus_sessions_status     ON focus_sessions(status);
CREATE INDEX IF NOT EXISTS idx_focus_sessions_started_at ON focus_sessions(started_at);

CREATE TRIGGER update_focus_sessions_updated_at BEFORE UPDATE ON focus_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Streaks table
CREATE TABLE IF NOT EXISTS streaks (
    id                SERIAL PRIMARY KEY,
    user_id           INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    current_streak    INT NOT NULL DEFAULT 0,
    longest_streak    INT NOT NULL DEFAULT 0,
    last_session_date DATE,
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_streaks_user_id ON streaks(user_id);

-- Badge definitions table
CREATE TABLE IF NOT EXISTS badge_definitions (
    id             SERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL UNIQUE,
    description    TEXT,
    icon_key       VARCHAR(100),
    criteria_type  VARCHAR(50) NOT NULL, -- streak | sessions | focus_time | jobs
    criteria_value INT NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

-- User earned badges table
CREATE TABLE IF NOT EXISTS user_badges (
    id        SERIAL PRIMARY KEY,
    user_id   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id  INT NOT NULL REFERENCES badge_definitions(id) ON DELETE CASCADE,
    earned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, badge_id)
);

CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);

-- Daily progress table
CREATE TABLE IF NOT EXISTS daily_progress (
    id                  SERIAL PRIMARY KEY,
    user_id             INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date                DATE NOT NULL,
    total_focus_seconds INT NOT NULL DEFAULT 0,
    sessions_completed  INT NOT NULL DEFAULT 0,
    UNIQUE (user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_progress_user_id ON daily_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_progress_date    ON daily_progress(date);

-- ============================================================
-- Notifications
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(50) NOT NULL,  -- session_completed | streak_milestone | badge_earned | daily_reminder | streak_warning
    title      VARCHAR(200) NOT NULL,
    message    TEXT NOT NULL,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id    ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read    ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);

-- ============================================================
-- Seed badge definitions
-- ============================================================
INSERT INTO badge_definitions (name, description, icon_key, criteria_type, criteria_value) VALUES
    ('First Lock-in',  'Complete your first focus session',       'first_session', 'sessions',   1),
    ('3-Day Streak',   'Maintain a 3-day streak',                 'streak_3',      'streak',     3),
    ('7-Day Streak',   'Maintain a 7-day streak',                 'streak_7',      'streak',     7),
    ('30-Day Streak',  'Maintain a 30-day streak',                'streak_30',     'streak',    30),
    ('Focus Rookie',   'Accumulate 1 hour of total focus time',   'focus_1h',      'focus_time', 3600),
    ('Focus Pro',      'Accumulate 10 hours of total focus time', 'focus_10h',     'focus_time', 36000),
    ('Job Hunter',     'Track 5 job applications',                'job_hunter',    'jobs',       5),
    ('Consistency',    'Complete 10 focus sessions',              'sessions_10',   'sessions',   10)
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- Serp Job Cache
-- ============================================================

CREATE TABLE IF NOT EXISTS serp_job_cache (
    id               SERIAL PRIMARY KEY,
    user_id          INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id      TEXT NOT NULL,
    title            TEXT NOT NULL,
    company_name     TEXT,
    location         TEXT,
    description      TEXT,
    link             TEXT,
    platform         VARCHAR(100),
    posted_at        VARCHAR(100),
    salary           VARCHAR(100),
    saved_to_tracker BOOLEAN NOT NULL DEFAULT FALSE,
    tracker_job_id   INT REFERENCES jobs(id) ON DELETE SET NULL,
    fetched_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_serp_job_cache_user_id    ON serp_job_cache(user_id);
CREATE INDEX IF NOT EXISTS idx_serp_job_cache_fetched_at ON serp_job_cache(fetched_at);
