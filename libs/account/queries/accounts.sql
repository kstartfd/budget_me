-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: GetAccountByTelegramID :one
SELECT * FROM accounts WHERE telegram_id = $1;

-- name: GetAccountByEmail :one
SELECT * FROM accounts WHERE email = $1;

-- name: InsertAccount :one
INSERT INTO accounts (
    id, telegram_id, email, password_hash,
    first_name, username, language_code,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpdateAccountTelegramProfile :one
UPDATE accounts
SET first_name = $2, username = $3, language_code = $4, updated_at = $5
WHERE id = $1
RETURNING *;

-- name: SetAccountWebIdentity :one
UPDATE accounts
SET email = $2, password_hash = $3, updated_at = $4
WHERE id = $1
RETURNING *;

-- name: RequestAccountDeletion :one
-- COALESCE keeps the first deadline: asking twice cannot push it back.
UPDATE accounts
SET deletion_requested_at = COALESCE(deletion_requested_at, sqlc.arg(now)),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CancelAccountDeletion :one
UPDATE accounts
SET deletion_requested_at = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RegisterAccountApp :exec
INSERT INTO account_apps (account_id, app)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListAccountsDueForPurge :many
-- Accounts whose grace period is over and which still hold data in this app.
SELECT a.id
FROM accounts a
JOIN account_apps aa ON aa.account_id = a.id
WHERE aa.app = sqlc.arg(app)
  AND a.deletion_requested_at IS NOT NULL
  AND a.deletion_requested_at <= sqlc.arg(cutoff);

-- name: UnregisterAccountApp :exec
DELETE FROM account_apps
WHERE app = sqlc.arg(app) AND account_id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteReleasedAccounts :execrows
-- The last app let go: nothing is left that the account could be needed for.
DELETE FROM accounts a
WHERE a.deletion_requested_at IS NOT NULL
  AND a.deletion_requested_at <= sqlc.arg(cutoff)
  AND NOT EXISTS (SELECT 1 FROM account_apps aa WHERE aa.account_id = a.id);

-- name: MoveAccountApps :exec
-- The survivor now holds data for every app the loser did.
INSERT INTO account_apps (account_id, app)
SELECT sqlc.arg(survivor)::uuid, app FROM account_apps WHERE account_id = sqlc.arg(loser)::uuid
ON CONFLICT DO NOTHING;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = $1;

-- name: InheritAccountIdentity :one
-- Fills only what the account lacks; what it has stays.
UPDATE accounts SET
    telegram_id   = COALESCE(telegram_id, sqlc.narg(telegram_id)),
    email         = COALESCE(email, sqlc.narg(email)),
    password_hash = COALESCE(password_hash, sqlc.narg(password_hash)),
    username      = COALESCE(username, sqlc.narg(username)),
    language_code = COALESCE(language_code, sqlc.narg(language_code)),
    updated_at    = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;
