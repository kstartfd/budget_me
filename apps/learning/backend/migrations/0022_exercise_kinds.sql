-- +goose Up
-- Three more ways to practise: writing a few sentences (the agent checks the
-- text), listening (a sentence read aloud and a question about it) and saying a
-- phrase aloud (the browser hears it and the server compares it).
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'write';
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'listen';
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'speak';

-- +goose Down
-- A value cannot be dropped from an enum; the extra kinds stay and are harmless.
SELECT 1;
