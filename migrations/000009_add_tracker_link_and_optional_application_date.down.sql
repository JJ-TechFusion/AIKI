DROP INDEX IF EXISTS idx_serp_job_cache_tracker_job_id;

ALTER TABLE serp_job_cache
    DROP COLUMN IF EXISTS tracker_job_id;

ALTER TABLE jobs
    ALTER COLUMN date_applied SET DEFAULT NOW(),
    ALTER COLUMN date_applied SET NOT NULL;
