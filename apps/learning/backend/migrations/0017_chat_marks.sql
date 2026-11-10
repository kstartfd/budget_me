-- +goose Up
CREATE TABLE learning_chat_marks (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    any_label boolean NOT NULL,
    dialogue boolean NOT NULL,
    triggers jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE learning_chat_marks;
