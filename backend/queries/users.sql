-- name: InsertUser :one
INSERT INTO users (
    id, telegram_id, email, password_hash,
    first_name, username, language_code,
    month_start_day, currency, timezone,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserTelegramProfile :one
UPDATE users
SET first_name = $2, username = $3, language_code = $4, updated_at = $5
WHERE id = $1
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE users
SET month_start_day = sqlc.arg(month_start_day),
    currency        = sqlc.arg(currency),
    updated_at      = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING month_start_day, currency;

-- name: RememberTimezone :exec
-- The zone the client last reported. Only background work reads it (cmd/worker);
-- requests carry their own zone with them.
UPDATE users SET timezone = sqlc.arg(timezone)
WHERE id = sqlc.arg(id) AND timezone <> sqlc.arg(timezone);

-- name: ListUsersForAdvice :many
-- Everyone the nightly advice pass should look at; their zone decides which
-- instants make up "their" month.
SELECT id, timezone FROM users
WHERE deletion_requested_at IS NULL;

-- name: SetUserWebIdentity :one
UPDATE users
SET email = $2, password_hash = $3, updated_at = $4
WHERE id = $1
RETURNING *;

-- name: RequestUserDeletion :one
-- Idempotent on purpose: asking twice keeps the ORIGINAL deadline (COALESCE),
-- so a second tap cannot silently push the deletion further out.
UPDATE users
SET deletion_requested_at = COALESCE(deletion_requested_at, sqlc.arg(now)),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CancelUserDeletion :one
UPDATE users
SET deletion_requested_at = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUsersPastDeadline :many
-- The worker's sweep: everything requested before the cutoff goes for good.
-- Returns the ids it removed so the run can be logged. All owned rows follow
-- via ON DELETE CASCADE.
DELETE FROM users
WHERE deletion_requested_at IS NOT NULL
  AND deletion_requested_at < sqlc.arg(cutoff)
RETURNING id, telegram_id, email;


-- name: SetBalanceAnchor :one
-- Re-stating the balance moves the anchor to `now`: transactions before it stop
-- counting, which is exactly what "my balance is actually X" should mean.
UPDATE users
SET balance_anchor_minor = sqlc.arg(balance_minor),
    balance_anchor_at    = sqlc.arg(now),
    updated_at           = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;
