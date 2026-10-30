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

-- name: ListLearningChatUsage :many
SELECT scope, tokens FROM learning_chat_usage
WHERE day = sqlc.arg(day)
  AND ((scope = 'user' AND subject = sqlc.arg(user_subject)) OR (scope = 'key' AND subject = sqlc.arg(key_subject)));

-- name: AddLearningChatUsage :exec
INSERT INTO learning_chat_usage (scope, subject, day, tokens, turns)
VALUES ($1, $2, $3, $4, 1)
ON CONFLICT (scope, subject, day) DO UPDATE SET
    tokens = learning_chat_usage.tokens + EXCLUDED.tokens,
    turns = learning_chat_usage.turns + 1;

-- name: DeleteOldLearningChatUsage :exec
DELETE FROM learning_chat_usage WHERE day < $1;

-- name: GetOpenTopicChat :one
SELECT * FROM learning_chats
WHERE topic_id = $1 AND user_id = $2 AND archived_at IS NULL;

-- name: EnsureOpenTopicChat :one
INSERT INTO learning_chats (id, user_id, topic_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (topic_id) WHERE topic_id IS NOT NULL AND archived_at IS NULL
DO UPDATE SET topic_id = EXCLUDED.topic_id
RETURNING *;

-- The next dialog carries over what the closed one spent in its budget window,
-- so clearing a chat never gives its budget back.
-- name: OpenNextTopicChat :exec
INSERT INTO learning_chats (id, user_id, topic_id, spent_tokens, spent_since, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6);

-- name: ArchiveLearningChat :exec
UPDATE learning_chats SET archived_at = $2, updated_at = $2, busy_until = NULL WHERE id = $1;

-- name: ListArchivedTopicChats :many
SELECT c.id, c.created_at, c.archived_at,
    COALESCE((
        SELECT left(m.content, 160) FROM learning_chat_messages m
        WHERE m.chat_id = c.id AND m.role = 'user' ORDER BY m.id LIMIT 1
    ), '')::text AS first_message,
    (SELECT count(*) FROM learning_chat_messages m WHERE m.chat_id = c.id)::bigint AS message_count
FROM learning_chats c
WHERE c.topic_id = $1 AND c.user_id = $2 AND c.archived_at IS NOT NULL
ORDER BY c.archived_at DESC, c.id
LIMIT 100;

-- name: GetArchivedTopicChat :one
SELECT * FROM learning_chats
WHERE id = $1 AND user_id = $2 AND topic_id IS NOT NULL AND archived_at IS NOT NULL;

-- name: DeleteArchivedTopicChat :execrows
DELETE FROM learning_chats
WHERE id = $1 AND user_id = $2 AND topic_id IS NOT NULL AND archived_at IS NOT NULL;
