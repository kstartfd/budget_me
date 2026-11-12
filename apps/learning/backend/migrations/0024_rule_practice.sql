-- +goose Up
-- Practice on one rule of the rule book: sentences and exercises an agent wrote
-- for this learner, and the chats about those sentences. It is a sitting, not a
-- record: leaving the rule's page puts a short expiry on it, coming back lifts
-- the expiry, and once it has passed the sitting is gone with its chats. The
-- expiry is always set (an app killed by the phone never says it has left), so
-- nothing outlives a day or so of silence.
CREATE TABLE learning_rule_practice (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_id text NOT NULL,
    -- What the agent is told about the rule and the learner when it writes
    -- more: kept with the sitting so a chat about a sentence needs no library.
    language text NOT NULL,
    translation_language text NOT NULL,
    level text NOT NULL,
    title text NOT NULL,
    theory text NOT NULL,
    -- [{"id","text","translation"}], and the exercise blocks, whose answer keys
    -- never leave the server until the learner asks for them.
    examples jsonb NOT NULL DEFAULT '[]'::jsonb,
    exercises jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    UNIQUE (user_id, rule_id)
);
CREATE INDEX learning_rule_practice_expires ON learning_rule_practice(expires_at);

-- The chat of one sentence of a sitting: a fourth kind of subject next to an
-- entry, a notebook and a program topic. It goes with the sitting.
ALTER TABLE learning_chats
    ADD COLUMN practice_id uuid REFERENCES learning_rule_practice(id) ON DELETE CASCADE,
    ADD COLUMN example_id text,
    DROP CONSTRAINT learning_chats_one_subject,
    ADD CONSTRAINT learning_chats_one_subject CHECK (num_nonnulls(entry_id, topic_id, path_topic_id, practice_id) = 1),
    ADD CONSTRAINT learning_chats_example CHECK ((practice_id IS NULL) = (example_id IS NULL));
CREATE UNIQUE INDEX learning_chats_example ON learning_chats(practice_id, example_id)
    WHERE practice_id IS NOT NULL;

-- +goose Down
DROP INDEX learning_chats_example;
DELETE FROM learning_chats WHERE practice_id IS NOT NULL;
ALTER TABLE learning_chats
    DROP CONSTRAINT learning_chats_example,
    DROP CONSTRAINT learning_chats_one_subject,
    DROP COLUMN example_id,
    DROP COLUMN practice_id,
    ADD CONSTRAINT learning_chats_one_subject CHECK (num_nonnulls(entry_id, topic_id, path_topic_id) = 1);
DROP TABLE learning_rule_practice;
