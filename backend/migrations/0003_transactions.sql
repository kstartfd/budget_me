-- +goose Up
CREATE TABLE transactions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id  uuid REFERENCES categories(id) ON DELETE SET NULL,
    amount_minor bigint NOT NULL,
    title        text   NOT NULL,
    spent_at     timestamptz NOT NULL,
    created_at   timestamptz NOT NULL
);

CREATE INDEX transactions_user_spent_at_idx ON transactions (user_id, spent_at);

-- +goose Down
DROP TABLE transactions;