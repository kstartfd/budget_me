-- +goose Up
-- A free chat can be a role-play: the learner picks a scenario (ordering food,
-- travel, shopping, a job interview) before the first message and the teacher
-- plays the other side. Fixed for the life of the dialog, so the system prompt
-- stays a constant prefix and the provider's cache keeps working.
ALTER TABLE learning_chats ADD COLUMN scenario text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE learning_chats DROP COLUMN scenario;
