-- +goose Up
-- A lesson is a timed 30-minute sitting on one topic of the program. The clock
-- never stops once started: pausing only marks the learner as away, so the
-- time spent paused is kept and the end moment stays where it was set.
-- Attempts made during a lesson point at it, so the history of what was
-- studied, for how long and how well can be read back later.
CREATE TYPE learning_path_lesson_status AS ENUM ('active', 'finished');

CREATE TABLE learning_path_lessons (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id uuid NOT NULL REFERENCES learning_path_topics(id) ON DELETE CASCADE,
    status learning_path_lesson_status NOT NULL,
    started_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    finished_at timestamptz,
    paused_at timestamptz,
    paused_seconds integer NOT NULL DEFAULT 0,
    pause_count integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX learning_path_lessons_one_active ON learning_path_lessons(user_id)
    WHERE status = 'active';
CREATE INDEX learning_path_lessons_topic ON learning_path_lessons(user_id, topic_id, started_at DESC);

ALTER TABLE learning_path_attempts
    ADD COLUMN lesson_id uuid REFERENCES learning_path_lessons(id) ON DELETE SET NULL;
CREATE INDEX learning_path_attempts_lesson ON learning_path_attempts(lesson_id) WHERE lesson_id IS NOT NULL;

-- The chat of a program topic: a third kind of subject next to an entry and a
-- notebook, with one never-archived conversation per topic.
ALTER TABLE learning_chats
    ADD COLUMN path_topic_id uuid REFERENCES learning_path_topics(id) ON DELETE CASCADE,
    DROP CONSTRAINT learning_chats_one_subject,
    ADD CONSTRAINT learning_chats_one_subject CHECK (num_nonnulls(entry_id, topic_id, path_topic_id) = 1);
CREATE UNIQUE INDEX learning_chats_path_topic ON learning_chats(path_topic_id)
    WHERE path_topic_id IS NOT NULL;

-- The program is a syllabus, not a pile of exercises: every topic belongs to a
-- unit (a stage of the program), carries the rule taught in plain words and
-- the key words to learn. Exercises, the chat and lessons all hang off it.
ALTER TABLE learning_path_topics
    ADD COLUMN unit text NOT NULL DEFAULT '',
    ADD COLUMN theory text NOT NULL DEFAULT '',
    ADD COLUMN vocabulary jsonb NOT NULL DEFAULT '[]'::jsonb;

-- What the learner told before the program was made: interests, the variant of
-- the language, the goal and how much time they have. The program is built
-- around it and it stays with the program for later analysis.
ALTER TABLE learning_paths
    ADD COLUMN preferences jsonb NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE learning_paths DROP COLUMN preferences;
ALTER TABLE learning_path_topics
    DROP COLUMN vocabulary,
    DROP COLUMN theory,
    DROP COLUMN unit;
DROP INDEX learning_chats_path_topic;
DELETE FROM learning_chats WHERE path_topic_id IS NOT NULL;
ALTER TABLE learning_chats
    DROP CONSTRAINT learning_chats_one_subject,
    DROP COLUMN path_topic_id,
    ADD CONSTRAINT learning_chats_one_subject CHECK ((entry_id IS NULL) <> (topic_id IS NULL));
DROP INDEX learning_path_attempts_lesson;
ALTER TABLE learning_path_attempts DROP COLUMN lesson_id;
DROP TABLE learning_path_lessons;
DROP TYPE learning_path_lesson_status;
