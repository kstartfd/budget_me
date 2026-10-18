-- name: InsertUser :one
INSERT INTO users (
    id, account_id, telegram_id, email, password_hash,
    first_name, username, language_code,
    month_start_day, currency, timezone,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;

-- name: GetUserByAccountID :one
SELECT * FROM users WHERE account_id = $1;

-- name: SetUserAccount :one
UPDATE users SET account_id = $2, updated_at = $3 WHERE id = $1 RETURNING *;

-- name: MirrorAccountToUser :one
-- The account is the source of truth for identity; this row keeps a copy.
UPDATE users
SET telegram_id = $2, email = $3, password_hash = $4,
    first_name = $5, username = $6, language_code = $7, updated_at = $8
WHERE id = $1 RETURNING *;

-- name: ListUsersWithoutAccount :many
SELECT * FROM users WHERE account_id IS NULL ORDER BY created_at;

-- name: DeleteUsersByAccount :exec
-- The worker's sweep for accounts whose deletion is due: all owned rows follow
-- via ON DELETE CASCADE.
DELETE FROM users WHERE account_id = ANY(sqlc.arg(account_ids)::uuid[]);

-- name: UpdateUserSettings :one
UPDATE users
SET month_start_day = sqlc.arg(month_start_day),
    currency        = sqlc.arg(currency),
    updated_at      = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING month_start_day, currency;

-- name: UpdateUserSettingsAndResetHistory :one
-- The same update, plus the line the running totals restart from. Moving the day
-- a period opens re-cuts every period ever shown, so a carry-over computed under
-- the old boundaries would be carried into periods that no longer match it.
--
-- The balance anchor goes with it: "right now I have X" was stated against the
-- old cut, and NULL is already the app's "never stated" — so it asks again
-- instead of showing a figure the user never confirmed. Transactions are left
-- alone; only what reads them forward changes.
--
-- One statement rather than an update followed by a reset: the two must not be
-- separable, and this keeps it atomic without reaching for a transaction.
UPDATE users
SET month_start_day      = sqlc.arg(month_start_day),
    currency             = sqlc.arg(currency),
    history_reset_at     = sqlc.arg(now),
    balance_anchor_minor = NULL,
    balance_anchor_at    = NULL,
    updated_at           = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING month_start_day, currency;

-- name: GetHistoryResetAt :one
-- Just the cutoff. The dashboard asks on every load and needs nothing else off
-- the user row, so this stays narrower than GetUserByID.
SELECT history_reset_at FROM users WHERE id = sqlc.arg(id);

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

-- name: SetBalanceAnchor :one
-- Re-stating the balance moves the anchor to `now`: transactions before it stop
-- counting, which is exactly what "my balance is actually X" should mean.
UPDATE users
SET balance_anchor_minor = sqlc.arg(balance_minor),
    balance_anchor_at    = sqlc.arg(now),
    updated_at           = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;
