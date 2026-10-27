-- name: GetLearningChatByEntry :one
SELECT * FROM learning_chats WHERE entry_id = $1 AND user_id = $2;

-- name: EnsureLearningChat :one
INSERT INTO learning_chats (id, user_id, entry_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (entry_id) DO UPDATE SET entry_id = EXCLUDED.entry_id
RETURNING *;

-- name: LockLearningChat :one
UPDATE learning_chats SET busy_until = sqlc.arg(busy_until)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
  AND (busy_until IS NULL OR busy_until < sqlc.arg(now))
RETURNING *;

-- name: UnlockLearningChat :exec
UPDATE learning_chats SET busy_until = NULL WHERE id = $1;

-- name: ListLearningChatMessages :many
SELECT * FROM learning_chat_messages WHERE chat_id = $1 ORDER BY id;

-- name: ListLearningChatMessagesAfter :many
SELECT * FROM learning_chat_messages WHERE chat_id = $1 AND id > $2 ORDER BY id;

-- name: InsertLearningChatMessage :one
INSERT INTO learning_chat_messages (chat_id, role, content, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- Records a finished turn: the size the next one starts from, and what it cost.
-- A budget window that has run out starts over with this turn (every SET
-- expression reads the row as it was, so both CASEs see the same spent_since).
-- name: RecordLearningChatTurn :exec
UPDATE learning_chats SET
    context_tokens = sqlc.arg(context_tokens),
    updated_at = sqlc.arg(now),
    spent_tokens = CASE
        WHEN spent_since IS NULL OR spent_since + make_interval(secs => sqlc.arg(window_secs)::float8) <= sqlc.arg(now)
        THEN 0 ELSE spent_tokens END + sqlc.arg(spent)::bigint,
    spent_since = CASE
        WHEN spent_since IS NULL OR spent_since + make_interval(secs => sqlc.arg(window_secs)::float8) <= sqlc.arg(now)
        THEN sqlc.arg(now) ELSE spent_since END
WHERE id = sqlc.arg(id);

-- name: SaveLearningChatSummary :exec
UPDATE learning_chats SET summary = $2, summary_upto = $3, context_tokens = 0, updated_at = $4 WHERE id = $1;

-- name: DeleteLearningChatMessages :exec
DELETE FROM learning_chat_messages WHERE chat_id = $1;

-- name: ResetLearningChat :exec
UPDATE learning_chats SET summary = '', summary_upto = 0, context_tokens = 0, updated_at = $2 WHERE id = $1;
