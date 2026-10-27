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

-- name: TouchLearningChat :exec
UPDATE learning_chats SET context_tokens = $2, updated_at = $3 WHERE id = $1;

-- name: SaveLearningChatSummary :exec
UPDATE learning_chats SET summary = $2, summary_upto = $3, context_tokens = 0, updated_at = $4 WHERE id = $1;

-- name: DeleteLearningChatMessages :exec
DELETE FROM learning_chat_messages WHERE chat_id = $1;

-- name: ResetLearningChat :exec
UPDATE learning_chats SET summary = '', summary_upto = 0, context_tokens = 0, updated_at = $2 WHERE id = $1;
