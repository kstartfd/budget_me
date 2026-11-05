-- name: ListLearningTopics :many
SELECT * FROM learning_topics WHERE user_id = $1 ORDER BY created_at, id;

-- name: GetLearningTopic :one
SELECT * FROM learning_topics WHERE id = $1 AND user_id = $2;

-- name: InsertLearningTopic :one
INSERT INTO learning_topics (id, user_id, name, language, translation_language, created_at)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: UpdateLearningTopic :one
UPDATE learning_topics SET name = $3, language = $4, translation_language = $5
WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteLearningTopic :execrows
DELETE FROM learning_topics WHERE id = $1 AND user_id = $2;

-- name: ListLearningEntries :many
SELECT * FROM learning_entries WHERE user_id = $1 AND topic_id = $2
AND (sqlc.arg(kind_filter)::text = '' OR kind::text = sqlc.arg(kind_filter))
AND (sqlc.arg(status_filter)::text = '' OR status::text = sqlc.arg(status_filter))
AND (sqlc.arg(search)::text = '' OR strpos(lower(text || ' ' || translation || ' ' || explanation || ' ' || rule || ' ' || source), lower(sqlc.arg(search))) > 0)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetLearningEntry :one
SELECT * FROM learning_entries WHERE id = $1 AND user_id = $2;

-- name: InsertLearningEntry :one
INSERT INTO learning_entries (id, user_id, topic_id, kind, text, text_hash, transcription, translation, explanation, native_usage, example, rule, source, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (user_id, topic_id, text_hash) DO UPDATE SET
kind = EXCLUDED.kind, text = EXCLUDED.text, transcription = EXCLUDED.transcription,
translation = EXCLUDED.translation, explanation = EXCLUDED.explanation, native_usage = EXCLUDED.native_usage,
example = EXCLUDED.example, rule = EXCLUDED.rule, source = EXCLUDED.source, status = EXCLUDED.status,
updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: UpdateLearningEntry :one
UPDATE learning_entries SET topic_id = $3, kind = $4, text = $5, text_hash = $6, transcription = $7,
translation = $8, explanation = $9, native_usage = $10, example = $11, rule = $12, source = $13, status = $14, updated_at = $15
WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteLearningEntry :execrows
DELETE FROM learning_entries WHERE id = $1 AND user_id = $2;

-- name: EnqueueLearningCapture :one
INSERT INTO learning_captures (id, user_id, request_key, payload, status, attempts, available_at, error, created_at, updated_at)
VALUES ($1, $2, $3, $4, 'pending', 0, $5, '', $5, $5)
ON CONFLICT (user_id, request_key) DO UPDATE SET request_key = learning_captures.request_key
RETURNING *;

-- name: GetLearningCapture :one
SELECT * FROM learning_captures WHERE id = $1 AND user_id = $2;

-- name: GetLearningCaptureByKey :one
SELECT * FROM learning_captures WHERE user_id = $1 AND request_key = $2;

-- name: ClaimLearningCapture :one
WITH next_job AS (
    SELECT id FROM learning_captures
    WHERE (status = 'pending' AND available_at <= sqlc.arg(now))
       OR (status = 'processing' AND locked_until <= sqlc.arg(now))
    ORDER BY available_at, created_at
    FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE learning_captures SET status = 'processing', attempts = attempts + 1,
    locked_until = sqlc.arg(locked_until), lease_id = sqlc.arg(lease_id), updated_at = sqlc.arg(now)
WHERE id = (SELECT id FROM next_job) RETURNING *;

-- name: LockLearningCaptureLease :one
SELECT * FROM learning_captures WHERE id = $1 AND lease_id = $2 AND status = 'processing' FOR UPDATE;

-- name: CompleteLearningCapture :exec
UPDATE learning_captures SET status = 'done', entry_id = $3, locked_until = NULL,
    lease_id = NULL, error = '', updated_at = $4
WHERE id = $1 AND lease_id = $2;

-- name: RetryLearningCapture :exec
UPDATE learning_captures SET status = $3, available_at = $4, locked_until = NULL,
    lease_id = NULL, error = $5, updated_at = $6 WHERE id = $1 AND lease_id = $2;

-- name: GetLearningAudioCache :one
SELECT * FROM learning_audio_cache WHERE language = $1 AND text_hash = $2;

-- name: InsertLearningAudioCache :one
INSERT INTO learning_audio_cache (id, language, text, text_hash, content_hash, storage_path,
    content_type, byte_size, provider, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
ON CONFLICT (language, text_hash) DO UPDATE SET last_used_at = EXCLUDED.last_used_at
RETURNING *;
