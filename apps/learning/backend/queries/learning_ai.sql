-- name: GetLearningAISettings :one
SELECT * FROM learning_ai_settings WHERE user_id = $1;

-- name: UpsertLearningAISettings :one
INSERT INTO learning_ai_settings (user_id, provider, model, level, teacher, key_ciphertext, key_hint, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    model = EXCLUDED.model,
    level = EXCLUDED.level,
    teacher = EXCLUDED.teacher,
    key_ciphertext = EXCLUDED.key_ciphertext,
    key_hint = EXCLUDED.key_hint,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: UpdateLearningAIChoice :one
UPDATE learning_ai_settings SET model = $3, level = $4, teacher = $5, updated_at = $6
WHERE user_id = $1 AND provider = $2
RETURNING *;

-- name: DeleteLearningAISettings :execrows
DELETE FROM learning_ai_settings WHERE user_id = $1;

-- name: KnownLearningTextHashes :many
SELECT DISTINCT text_hash FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND text_hash = ANY(sqlc.arg(hashes)::bytea[]);

-- name: RecordLearningScan :execrows
INSERT INTO learning_scans (user_id, topic_id, image_hash, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;
