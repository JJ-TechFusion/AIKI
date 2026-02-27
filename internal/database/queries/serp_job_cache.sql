-- name: UpsertSerpJobCache :one
INSERT INTO serp_job_cache (
    user_id,
    external_id,
    title,
    company_name,
    location,
    description,
    link,
    platform,
    posted_at,
    salary
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) ON CONFLICT (user_id, external_id) DO UPDATE SET
    title        = EXCLUDED.title,
    company_name = EXCLUDED.company_name,
    location     = EXCLUDED.location,
    description  = EXCLUDED.description,
    link         = EXCLUDED.link,
    platform     = EXCLUDED.platform,
    posted_at    = EXCLUDED.posted_at,
    salary       = EXCLUDED.salary,
    fetched_at   = NOW()
RETURNING *;

-- name: GetCachedJobsByUserID :many
SELECT * FROM serp_job_cache
WHERE user_id = $1
ORDER BY fetched_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCachedJobByID :one
SELECT * FROM serp_job_cache
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: GetLatestCacheFetchTime :one
SELECT fetched_at FROM serp_job_cache
WHERE user_id = $1
ORDER BY fetched_at DESC
LIMIT 1;

-- name: MarkJobSavedToTracker :exec
UPDATE serp_job_cache
SET saved_to_tracker = TRUE
WHERE id = $1 AND user_id = $2;

-- name: DeleteOldCacheForUser :exec
DELETE FROM serp_job_cache
WHERE user_id = $1
  AND fetched_at < NOW() - INTERVAL '24 hours';