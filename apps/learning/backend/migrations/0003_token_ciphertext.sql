-- +goose Up
ALTER TABLE learning_tokens ADD COLUMN token_ciphertext bytea;

-- +goose Down
ALTER TABLE learning_tokens DROP COLUMN token_ciphertext;
