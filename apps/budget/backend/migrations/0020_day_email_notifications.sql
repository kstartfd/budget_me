-- +goose Up
CREATE TYPE day_email_frequency AS ENUM ('daily', 'twice');
CREATE TYPE day_email_slot AS ENUM ('morning', 'evening');

-- A separate confirmation is required: an account email can be added without
-- proving ownership, while task titles must only go to an opted-in recipient.
CREATE TABLE day_email_subscriptions (
    user_id                 uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email                   text NOT NULL,
    frequency               day_email_frequency NOT NULL,
    confirmed_at            timestamptz,
    confirmation_hash       text UNIQUE,
    confirmation_expires_at timestamptz,
    created_at              timestamptz NOT NULL,
    updated_at              timestamptz NOT NULL
);

-- Freeze each digest before sending. Retries use the same body and Resend
-- idempotency key, even if a task changes between worker runs.
CREATE TABLE day_email_deliveries (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    local_day  date NOT NULL,
    slot       day_email_slot NOT NULL,
    email      text NOT NULL,
    subject    text NOT NULL,
    body       text NOT NULL,
    resend_id  text,
    sent_at    timestamptz,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, local_day, slot)
);

-- +goose Down
DROP TABLE day_email_deliveries;
DROP TABLE day_email_subscriptions;
DROP TYPE day_email_slot;
DROP TYPE day_email_frequency;
