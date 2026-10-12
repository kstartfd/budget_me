-- +goose Up
ALTER TABLE learning_entries ADD COLUMN source text NOT NULL DEFAULT '';
ALTER TABLE learning_entries ADD COLUMN explanation text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE learning_entries DROP COLUMN source;
ALTER TABLE learning_entries DROP COLUMN explanation;
