-- +goose Up
CREATE TEMPORARY TABLE learning_entry_merge ON COMMIT DROP AS
WITH ranked AS (
    SELECT id,
           first_value(id) OVER (
               PARTITION BY user_id, topic_id, text_hash
               ORDER BY updated_at DESC, created_at DESC, id DESC
           ) AS keep_id,
           row_number() OVER (
               PARTITION BY user_id, topic_id, text_hash
               ORDER BY updated_at DESC, created_at DESC, id DESC
           ) AS rank
    FROM learning_entries
)
SELECT id, keep_id, rank FROM ranked;

UPDATE learning_captures AS capture
SET entry_id = merge.keep_id
FROM learning_entry_merge AS merge
WHERE capture.entry_id = merge.id AND merge.rank > 1;

CREATE TEMPORARY TABLE learning_entry_chat_merge ON COMMIT DROP AS
SELECT chat.id AS chat_id, merge.keep_id,
       row_number() OVER (
           PARTITION BY merge.keep_id
           ORDER BY chat.updated_at DESC, chat.id DESC
       ) AS rank
FROM learning_chats AS chat
JOIN learning_entry_merge AS merge ON merge.id = chat.entry_id
WHERE merge.keep_id IN (SELECT keep_id FROM learning_entry_merge WHERE rank > 1);

UPDATE learning_chat_messages AS message
SET chat_id = survivor.chat_id
FROM learning_entry_chat_merge AS old_chat
JOIN learning_entry_chat_merge AS survivor
  ON survivor.keep_id = old_chat.keep_id AND survivor.rank = 1
WHERE message.chat_id = old_chat.chat_id AND old_chat.rank > 1;

DELETE FROM learning_chats AS chat
USING learning_entry_chat_merge AS merge
WHERE chat.id = merge.chat_id AND merge.rank > 1;

UPDATE learning_chats AS chat
SET entry_id = merge.keep_id,
    summary = '',
    summary_upto = 0,
    context_tokens = 0,
    busy_until = NULL
FROM learning_entry_chat_merge AS merge
WHERE chat.id = merge.chat_id AND merge.rank = 1;

DELETE FROM learning_entries AS entry
USING learning_entry_merge AS merge
WHERE entry.id = merge.id AND merge.rank > 1;

DROP INDEX learning_entries_topic_hash;
CREATE UNIQUE INDEX learning_entries_topic_hash ON learning_entries(user_id, topic_id, text_hash);

-- +goose Down
DROP INDEX learning_entries_topic_hash;
CREATE INDEX learning_entries_topic_hash ON learning_entries(user_id, topic_id, text_hash);
