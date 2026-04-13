-- Safety net if 000009 never ran or the table was dropped; matches post-000010 shape (TEXT columns).
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

CREATE INDEX IF NOT EXISTS idx_serp_job_cache_user_id ON serp_job_cache(user_id);
CREATE INDEX IF NOT EXISTS idx_serp_job_cache_fetched_at ON serp_job_cache(fetched_at);
CREATE INDEX IF NOT EXISTS idx_serp_job_cache_tracker_job_id ON serp_job_cache(tracker_job_id);
