-- name: CreateUserProfile :one
INSERT INTO user_profile (
    user_id,
    full_name,
    current_job,
    experience_level,
    goals
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUserProfileByUserID :one
SELECT * FROM user_profile
WHERE user_id = $1
LIMIT 1;

-- name: UpdateUserProfile :one
UPDATE user_profile
SET
    full_name = CASE WHEN sqlc.narg('full_name')::text IS NOT NULL THEN sqlc.narg('full_name') ELSE full_name END,
    current_job = CASE WHEN sqlc.narg('current_job')::text IS NOT NULL THEN sqlc.narg('current_job') ELSE current_job END,
    experience_level = CASE WHEN sqlc.narg('experience_level')::text IS NOT NULL THEN sqlc.narg('experience_level') ELSE experience_level END,
    goals = CASE WHEN sqlc.narg('goals')::text[] IS NOT NULL THEN sqlc.narg('goals') ELSE goals END,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;


-- name: UploadUserCV :one
UPDATE user_profile
SET
    cv = $2,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: GetUserCV :one
SELECT cv FROM user_profile
WHERE user_id = $1
LIMIT 1;