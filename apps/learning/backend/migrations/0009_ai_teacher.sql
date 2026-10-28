-- +goose Up
-- Whether the chat plays the language teacher: it talks in the language being
-- learned and corrects the learner's mistakes in their own. Off is a plain chat
-- about the saved entry.
ALTER TABLE learning_ai_settings
    ADD COLUMN teacher boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE learning_ai_settings
    DROP COLUMN teacher;
