ALTER TABLE user_profile
    ADD COLUMN IF NOT EXISTS job_search_location TEXT NOT NULL DEFAULT '';
