-- May fail if any row exceeds old limits; widen again with 000010 up migration if needed.
ALTER TABLE serp_job_cache
    ALTER COLUMN external_id TYPE VARCHAR(255),
    ALTER COLUMN title TYPE VARCHAR(255),
    ALTER COLUMN company_name TYPE VARCHAR(255),
    ALTER COLUMN location TYPE VARCHAR(255);
