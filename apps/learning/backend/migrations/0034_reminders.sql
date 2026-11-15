-- +goose Up
-- Opt-in daily reminder, sent by the Telegram bot when cards are due for review.
-- The learner's time zone comes from their browser when they switch it on, so
-- "morning" is theirs. last_sent_on (their local date) keeps it to one a day.
CREATE TABLE learning_reminders (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled boolean NOT NULL DEFAULT true,
    timezone text NOT NULL,
    last_sent_on date,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE learning_reminders;
