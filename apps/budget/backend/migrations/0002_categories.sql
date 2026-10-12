-- +goose Up
CREATE TABLE categories (
    id            uuid PRIMARY KEY,
    user_id       uuid REFERENCES users(id) ON DELETE CASCADE,
    slug          text,
    name          text,
    color         text NOT NULL,
    created_at    timestamptz NOT NULL,
    UNIQUE (slug)
);

-- +goose Down
DROP TABLE categories;