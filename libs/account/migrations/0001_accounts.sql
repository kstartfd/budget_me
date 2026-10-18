-- +goose Up
-- One account per person, shared by every app: identity (Telegram, email +
-- password) and the deletion request live here; each app keeps only its own data
-- and a users(id) row with the same id as the account.
CREATE TYPE app_name AS ENUM ('budget', 'learning');

CREATE TABLE accounts (
    id uuid PRIMARY KEY,
    telegram_id bigint UNIQUE,
    email text UNIQUE,
    password_hash text,
    first_name text NOT NULL,
    username text,
    language_code text,
    deletion_requested_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

-- Which apps hold data for the account. A deletion is carried out app by app;
-- the account row goes when the last app has let go of it.
CREATE TABLE account_apps (
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    app app_name NOT NULL,
    PRIMARY KEY (account_id, app)
);

-- +goose Down
DROP TABLE account_apps;
DROP TABLE accounts;
DROP TYPE app_name;
