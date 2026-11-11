-- +goose Up
-- The questions of a level test are written by the learner's agent, one batch
-- per level, so they have to be kept with the test: the answer key lives only
-- here and is never sent to the client.
CREATE TABLE learning_path_test_questions (
    test_id uuid NOT NULL REFERENCES learning_path_tests(id) ON DELETE CASCADE,
    question_key text NOT NULL,
    level text NOT NULL,
    position integer NOT NULL,
    skill text NOT NULL,
    text text NOT NULL DEFAULT '',
    prompt text NOT NULL,
    options jsonb NOT NULL,
    answer text NOT NULL,
    PRIMARY KEY (test_id, question_key)
);
CREATE INDEX learning_path_test_questions_level ON learning_path_test_questions(test_id, level, position);

-- +goose Down
DROP TABLE learning_path_test_questions;
