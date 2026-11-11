-- +goose Up
-- The library topic a program topic was made from, so its rule can be shown
-- with it. Empty for topics an agent wrote on its own; those are recognised by
-- their title.
ALTER TABLE learning_path_topics ADD COLUMN source_id text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE learning_path_topics DROP COLUMN source_id;
