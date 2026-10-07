-- +goose Up
ALTER TABLE learning_entries
    ADD COLUMN explanation text NOT NULL DEFAULT '',
    ADD COLUMN source text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE learning_entries
    DROP COLUMN source,
    DROP COLUMN explanation;
