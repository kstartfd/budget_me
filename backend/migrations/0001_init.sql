-- +goose Up
-- Users: one row per person, reachable via Telegram (telegram_id) and/or a
-- web account (email + password_hash). At least one identity is guaranteed
-- by the application layer (DB holds structure only — no CHECKs/defaults;
-- ids and timestamps are generated in Go).
CREATE TABLE users (
    id            uuid PRIMARY KEY,
    telegram_id   bigint UNIQUE,
    email         text UNIQUE,
    password_hash text,
    first_name    text NOT NULL,
    username      text,
    language_code text,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL
);

-- +goose Down
DROP TABLE users;
