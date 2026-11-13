-- +goose Up
-- How many mistakes the agent counted in a text the learner wrote on their own,
-- read the way a native speaker would read it. Zero for every other kind of
-- exercise, where an answer is simply right or wrong.
ALTER TABLE learning_path_attempts ADD COLUMN mistakes integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE learning_path_attempts DROP COLUMN mistakes;
