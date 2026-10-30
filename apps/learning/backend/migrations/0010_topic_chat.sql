-- +goose Up
-- A chat that belongs to a topic instead of an entry: a free conversation with
-- the language teacher. A topic has one open dialog at a time; clearing it
-- closes the dialog (archived_at) and opens a fresh one, so nothing said is lost.
-- Chats of an entry never have topic_id and are never archived.
ALTER TABLE learning_chats
    ALTER COLUMN entry_id DROP NOT NULL,
    ADD COLUMN topic_id uuid REFERENCES learning_topics(id) ON DELETE CASCADE,
    ADD COLUMN archived_at timestamptz,
    ADD CONSTRAINT learning_chats_one_subject CHECK ((entry_id IS NULL) <> (topic_id IS NULL));

CREATE UNIQUE INDEX learning_chats_open_topic ON learning_chats(topic_id)
    WHERE topic_id IS NOT NULL AND archived_at IS NULL;
CREATE INDEX learning_chats_topic_archive ON learning_chats(topic_id, archived_at DESC)
    WHERE archived_at IS NOT NULL;

-- +goose Down
DROP INDEX learning_chats_topic_archive;
DROP INDEX learning_chats_open_topic;
DELETE FROM learning_chats WHERE topic_id IS NOT NULL;
ALTER TABLE learning_chats
    DROP CONSTRAINT learning_chats_one_subject,
    DROP COLUMN archived_at,
    DROP COLUMN topic_id,
    ALTER COLUMN entry_id SET NOT NULL;
