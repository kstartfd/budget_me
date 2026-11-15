-- name: GetLearningReminder :one
SELECT * FROM learning_reminders WHERE user_id = $1;

-- name: UpsertLearningReminder :one
INSERT INTO learning_reminders (user_id, enabled, timezone, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (user_id) DO UPDATE SET enabled = EXCLUDED.enabled, timezone = EXCLUDED.timezone, updated_at = EXCLUDED.updated_at
RETURNING *;

-- Everyone the bot may write to: switched on, and signed in through Telegram.
-- name: ListEnabledLearningReminders :many
SELECT r.user_id, r.timezone, r.last_sent_on, u.telegram_id, u.language_code
FROM learning_reminders r
JOIN users u ON u.id = r.user_id
WHERE r.enabled AND u.telegram_id IS NOT NULL;

-- What is waiting for a user across all topics: seen before and due again, and never reviewed.
-- name: CountLearningDueForUser :one
SELECT count(*) FILTER (WHERE next_review_at IS NOT NULL)::bigint AS overdue,
       count(*) FILTER (WHERE next_review_at IS NULL)::bigint AS fresh
FROM learning_entries
WHERE user_id = $1 AND status <> 'known'
  AND (next_review_at IS NULL OR next_review_at <= $2);

-- name: MarkLearningReminderSent :exec
UPDATE learning_reminders SET last_sent_on = $2, updated_at = $3 WHERE user_id = $1;

-- A user who blocked the bot is switched off, so it is not tried again every hour.
-- name: DisableLearningReminder :exec
UPDATE learning_reminders SET enabled = false, updated_at = $2 WHERE user_id = $1;
