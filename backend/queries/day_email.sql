-- name: GetDayEmailSubscription :one
SELECT * FROM day_email_subscriptions WHERE user_id = sqlc.arg(user_id);

-- name: UpsertPendingDayEmailSubscription :one
INSERT INTO day_email_subscriptions (
    user_id, email, frequency, confirmation_hash, confirmation_expires_at,
    created_at, updated_at
) VALUES (
    sqlc.arg(user_id), sqlc.arg(email), sqlc.arg(frequency),
    sqlc.arg(confirmation_hash), sqlc.arg(confirmation_expires_at),
    sqlc.arg(now), sqlc.arg(now)
)
ON CONFLICT (user_id) DO UPDATE SET
    email = EXCLUDED.email,
    frequency = EXCLUDED.frequency,
    confirmed_at = NULL,
    confirmation_hash = EXCLUDED.confirmation_hash,
    confirmation_expires_at = EXCLUDED.confirmation_expires_at,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: UpdateConfirmedDayEmailFrequency :one
UPDATE day_email_subscriptions
SET frequency = sqlc.arg(frequency), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id)
  AND email = sqlc.arg(email)
  AND confirmed_at IS NOT NULL
RETURNING *;

-- name: ConfirmDayEmailSubscription :one
UPDATE day_email_subscriptions AS s
SET confirmed_at = sqlc.arg(now),
    confirmation_hash = NULL,
    confirmation_expires_at = NULL,
    updated_at = sqlc.arg(now)
WHERE s.confirmation_hash = sqlc.arg(confirmation_hash)
  AND s.user_id = sqlc.arg(user_id)
  AND s.confirmation_expires_at > sqlc.arg(now)
  AND EXISTS (
      SELECT 1 FROM users u
      WHERE u.id = s.user_id AND u.email = s.email
        AND u.deletion_requested_at IS NULL
  )
RETURNING *;

-- name: DeleteDayEmailSubscription :exec
DELETE FROM day_email_subscriptions WHERE user_id = sqlc.arg(user_id);

-- name: ListActiveDayEmailSubscriptions :many
SELECT s.user_id, s.email, s.frequency, s.confirmed_at, u.timezone,
       u.language_code
FROM day_email_subscriptions s
JOIN users u ON u.id = s.user_id
WHERE s.confirmed_at IS NOT NULL
  AND u.email = s.email
  AND u.deletion_requested_at IS NULL;

-- name: ListOpenTasksForDigest :many
SELECT title, planned_for FROM tasks
WHERE user_id = sqlc.arg(user_id)
  AND done_on IS NULL
  AND planned_for <= sqlc.arg(local_day)::date
ORDER BY planned_for, created_at;

-- name: InsertDayEmailDelivery :execrows
INSERT INTO day_email_deliveries (
    user_id, local_day, slot, email, subject, body, created_at
) VALUES (
    sqlc.arg(user_id), sqlc.arg(local_day), sqlc.arg(slot),
    sqlc.arg(email), sqlc.arg(subject), sqlc.arg(body), sqlc.arg(now)
)
ON CONFLICT (user_id, local_day, slot) DO NOTHING;

-- name: GetDayEmailDelivery :one
SELECT * FROM day_email_deliveries
WHERE user_id = sqlc.arg(user_id)
  AND local_day = sqlc.arg(local_day)
  AND slot = sqlc.arg(slot);

-- name: MarkDayEmailDeliverySent :exec
UPDATE day_email_deliveries
SET sent_at = sqlc.arg(now), resend_id = sqlc.arg(resend_id)
WHERE user_id = sqlc.arg(user_id)
  AND local_day = sqlc.arg(local_day)
  AND slot = sqlc.arg(slot)
  AND sent_at IS NULL;
