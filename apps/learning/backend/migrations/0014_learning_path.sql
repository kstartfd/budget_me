-- +goose Up
-- The teacher's level-by-level program: a placement/final test built from the
-- in-code question bank, the program the agent builds from its result, the
-- exercises it writes per topic and every attempt the learner makes. Nothing
-- here is ever deleted: resetting a path archives it, so the whole history
-- stays queryable.
CREATE TYPE learning_path_test_kind AS ENUM ('placement', 'final');
CREATE TYPE learning_path_test_status AS ENUM ('in_progress', 'done');
CREATE TYPE learning_path_status AS ENUM ('active', 'finished', 'abandoned');
CREATE TYPE learning_path_topic_status AS ENUM ('not_started', 'in_progress', 'completed');
CREATE TYPE learning_path_exercise_kind AS ENUM ('choice', 'fill', 'translate');

CREATE TABLE learning_paths (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language text NOT NULL,
    translation_language text NOT NULL,
    level text NOT NULL,
    summary text NOT NULL DEFAULT '',
    status learning_path_status NOT NULL,
    -- How the program came to be: the first analysis of this language, or a
    -- re-analysis after the learner reset the level.
    source text NOT NULL,
    -- The placement test the level was read off; no foreign key so a test and
    -- its path cannot deadlock on creation, and tests are never deleted anyway.
    placement_test_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    archived_at timestamptz
);
-- One live program per learner and language; starting another archives this.
CREATE UNIQUE INDEX learning_paths_one_active ON learning_paths(user_id, language)
    WHERE status = 'active';
CREATE INDEX learning_paths_history ON learning_paths(user_id, created_at DESC);

CREATE TABLE learning_path_tests (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path_id uuid REFERENCES learning_paths(id) ON DELETE CASCADE,
    kind learning_path_test_kind NOT NULL,
    language text NOT NULL,
    translation_language text NOT NULL,
    -- The level the ladder is asking about now; moves up while batches pass.
    level text NOT NULL,
    -- Result, written when the test is done: highest passed level, and, for a
    -- final test, whether the level was confirmed.
    level_reached text NOT NULL DEFAULT '',
    passed boolean NOT NULL DEFAULT false,
    status learning_path_test_status NOT NULL,
    created_at timestamptz NOT NULL,
    completed_at timestamptz
);
CREATE INDEX learning_path_tests_user ON learning_path_tests(user_id, created_at DESC);

CREATE TABLE learning_path_test_answers (
    test_id uuid NOT NULL REFERENCES learning_path_tests(id) ON DELETE CASCADE,
    question_key text NOT NULL,
    -- What the learner actually chose — the analyst reads the wrong picks.
    answer text NOT NULL DEFAULT '',
    correct boolean NOT NULL,
    PRIMARY KEY (test_id, question_key)
);

CREATE TABLE learning_path_topics (
    id uuid PRIMARY KEY,
    path_id uuid NOT NULL REFERENCES learning_paths(id) ON DELETE CASCADE,
    position integer NOT NULL,
    skill text NOT NULL,
    title text NOT NULL,
    goal text NOT NULL,
    status learning_path_topic_status NOT NULL,
    created_at timestamptz NOT NULL,
    completed_at timestamptz
);
CREATE INDEX learning_path_topics_path ON learning_path_topics(path_id, position, id);

CREATE TABLE learning_path_exercises (
    id uuid PRIMARY KEY,
    topic_id uuid NOT NULL REFERENCES learning_path_topics(id) ON DELETE CASCADE,
    position integer NOT NULL,
    kind learning_path_exercise_kind NOT NULL,
    -- A reading passage when the exercise has one, then the question itself.
    text text NOT NULL DEFAULT '',
    prompt text NOT NULL,
    options jsonb NOT NULL DEFAULT '[]'::jsonb,
    -- The key the deterministic grader compares against (choice/fill), or the
    -- reference translation the agent grades against (translate).
    answer text NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX learning_path_exercises_topic ON learning_path_exercises(topic_id, position, id);

CREATE TABLE learning_path_attempts (
    id uuid PRIMARY KEY,
    exercise_id uuid NOT NULL REFERENCES learning_path_exercises(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    answer text NOT NULL,
    correct boolean NOT NULL,
    -- What the agent said about the answer: the mistake explained in the
    -- learner's own language. Empty when the answer was right and needed none.
    feedback text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL
);
CREATE INDEX learning_path_attempts_exercise ON learning_path_attempts(exercise_id, created_at, id);

-- +goose Down
DROP TABLE learning_path_attempts;
DROP TABLE learning_path_exercises;
DROP TABLE learning_path_topics;
DROP TABLE learning_path_test_answers;
DROP TABLE learning_path_tests;
DROP TABLE learning_paths;
DROP TYPE learning_path_exercise_kind;
DROP TYPE learning_path_topic_status;
DROP TYPE learning_path_status;
DROP TYPE learning_path_test_status;
DROP TYPE learning_path_test_kind;
