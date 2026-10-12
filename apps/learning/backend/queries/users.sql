-- name: InsertUser :one
INSERT INTO users (id, telegram_id, email, password_hash, first_name, username, language_code, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserTelegramProfile :one
UPDATE users SET first_name = $2, username = $3, language_code = $4, updated_at = $5
WHERE id = $1 RETURNING *;

-- name: SetUserWebIdentity :one
UPDATE users SET email = $2, password_hash = $3, updated_at = $4
WHERE id = $1 RETURNING *;
