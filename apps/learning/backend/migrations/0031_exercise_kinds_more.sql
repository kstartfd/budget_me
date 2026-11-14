-- +goose Up
-- Three more ways to practise: putting the words of a sentence in order, a long
-- text with several gaps to fill, and a text with questions to answer.
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'order';
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'cloze';
ALTER TYPE learning_path_exercise_kind ADD VALUE IF NOT EXISTS 'read';

-- +goose Down
-- A value cannot be dropped from an enum; the extra kinds stay and are harmless.
SELECT 1;
