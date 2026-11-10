-- +goose Up
-- Each generated exercise carries the short rule that explains its answer.
-- Existing exercise sets keep working; their rule is empty and the client
-- falls back to the topic goal.
ALTER TABLE learning_path_exercises
    ADD COLUMN rule text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE learning_path_exercises
    DROP COLUMN rule;
