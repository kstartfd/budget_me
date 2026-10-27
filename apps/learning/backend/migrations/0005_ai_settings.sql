-- +goose Up
CREATE TYPE learning_ai_provider AS ENUM ('anthropic', 'openai', 'deepseek');
CREATE TYPE learning_ai_level AS ENUM ('none', 'off', 'low', 'medium', 'high', 'xhigh', 'max');

CREATE TABLE learning_ai_settings (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider learning_ai_provider NOT NULL,
    model text NOT NULL,
    level learning_ai_level NOT NULL,
    key_ciphertext bytea NOT NULL,
    key_hint text NOT NULL,
    updated_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE learning_ai_settings;
DROP TYPE learning_ai_level;
DROP TYPE learning_ai_provider;
