-- Table was previously only in schema.sql (sqlc); ensure it exists before ALTER.
CREATE TABLE IF NOT EXISTS serp_job_cache (
    id               SERIAL PRIMARY KEY,
    user_id          INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id      VARCHAR(255) NOT NULL,
    title            VARCHAR(255) NOT NULL,
    company_name     VARCHAR(255),
    location         VARCHAR(255),
    description      TEXT,
    link             TEXT,
    platform         VARCHAR(100),
    posted_at        VARCHAR(100),
    salary           VARCHAR(100),
    saved_to_tracker BOOLEAN NOT NULL DEFAULT FALSE,
    fetched_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_serp_job_cache_user_id ON serp_job_cache(user_id);
CREATE INDEX IF NOT EXISTS idx_serp_job_cache_fetched_at ON serp_job_cache(fetched_at);

ALTER TABLE jobs
    ALTER COLUMN date_applied DROP NOT NULL,
    ALTER COLUMN date_applied DROP DEFAULT;

ALTER TABLE serp_job_cache
    ADD COLUMN IF NOT EXISTS tracker_job_id INT REFERENCES jobs(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_serp_job_cache_tracker_job_id ON serp_job_cache(tracker_job_id);
