-- +goose Up
-- What a chat has cost within the current budget window: prompt plus reply tokens
-- of every turn. The window opens with the first turn after the previous one
-- ran out, so clearing a chat never gives its budget back.
ALTER TABLE learning_chats
    ADD COLUMN spent_tokens bigint NOT NULL DEFAULT 0,
    ADD COLUMN spent_since timestamptz;

-- +goose Down
ALTER TABLE learning_chats
    DROP COLUMN spent_since,
    DROP COLUMN spent_tokens;
