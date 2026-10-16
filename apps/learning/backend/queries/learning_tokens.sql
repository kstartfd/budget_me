-- name: InsertLearningToken :one
INSERT INTO learning_tokens (id, user_id, topic_id, name, prefix, token_hash, token_ciphertext, scope, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetLearningToken :one
SELECT * FROM learning_tokens WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: ListLearningTokens :many
SELECT * FROM learning_tokens WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC;

-- name: RevokeLearningToken :execrows
UPDATE learning_tokens SET revoked_at = $3 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: AuthenticateLearningToken :one
UPDATE learning_tokens SET last_used_at = sqlc.arg(now)
WHERE token_hash = sqlc.arg(token_hash) AND revoked_at IS NULL
RETURNING *;
