-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, message)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUnreadCount :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1 AND user_id = $2;

-- name: GetUsersWithNoSessionToday :many
SELECT DISTINCT u.id FROM users u
LEFT JOIN focus_sessions fs ON fs.user_id = u.id
    AND fs.status = 'completed'
    AND DATE(fs.ended_at) = CURRENT_DATE
WHERE u.is_active = TRUE
  AND fs.id IS NULL;