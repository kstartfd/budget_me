-- +goose Up
-- Which language a teacher explains in. By default it is the learner's own
-- language (translation_language); a learner who is past the first levels may
-- ask for the learned language instead, so rules, the program, exercises and
-- the explanation of a mistake are all written in it. The own language is kept
-- either way, to be switched back to.
ALTER TABLE learning_teachers ADD COLUMN explain_in_target boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE learning_teachers DROP COLUMN explain_in_target;
