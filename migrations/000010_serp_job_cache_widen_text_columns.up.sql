-- Google job_id / embedded payloads can exceed 255 chars; long titles also fail VARCHAR(255).
ALTER TABLE serp_job_cache
    ALTER COLUMN external_id TYPE TEXT,
    ALTER COLUMN title TYPE TEXT,
    ALTER COLUMN company_name TYPE TEXT,
    ALTER COLUMN location TYPE TEXT;
