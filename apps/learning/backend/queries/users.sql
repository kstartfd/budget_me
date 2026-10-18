-- name: InsertUser :one
INSERT INTO users (id, account_id, telegram_id, email, password_hash, first_name, username, language_code, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByAccountID :one
SELECT * FROM users WHERE account_id = $1;

-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;

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
DELETE FROM users WHERE account_id = ANY(sqlc.arg(account_ids)::uuid[]);
