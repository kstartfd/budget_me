-- +goose Up
-- Kimi (Moonshot AI) joins the agents a learner can connect.
ALTER TYPE learning_ai_provider ADD VALUE IF NOT EXISTS 'kimi';

-- +goose Down
-- An enum value cannot be dropped: the learners who picked Kimi lose their
-- agent settings, and the value stays unused.
DELETE FROM learning_ai_settings WHERE provider = 'kimi';
