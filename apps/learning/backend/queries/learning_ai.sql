-- name: GetLearningAISettings :one
SELECT * FROM learning_ai_settings WHERE user_id = $1;

-- name: UpsertLearningAISettings :one
INSERT INTO learning_ai_settings (user_id, provider, model, level, key_ciphertext, key_hint, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    model = EXCLUDED.model,
    level = EXCLUDED.level,
    key_ciphertext = EXCLUDED.key_ciphertext,
    key_hint = EXCLUDED.key_hint,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: UpdateLearningAIChoice :one
UPDATE learning_ai_settings SET model = $3, level = $4, updated_at = $5
WHERE user_id = $1 AND provider = $2
RETURNING *;

-- name: DeleteLearningAISettings :execrows
DELETE FROM learning_ai_settings WHERE user_id = $1;
