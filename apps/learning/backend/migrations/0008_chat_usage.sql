-- +goose Up
-- What the chat has cost per day, for limits that outlive any single chat:
--   scope 'user': a user's tokens across all their chats (subject = user id), so
--                 changing the API key or starting a new chat gives nothing back;
--   scope 'key':  one API key's tokens across every account using it (subject =
--                 the key's fingerprint, never the key).
-- `day` is the UTC date. Rows older than a week are swept as turns are recorded.
CREATE TABLE learning_chat_usage (
    scope text NOT NULL CHECK (scope IN ('user', 'key')),
    subject text NOT NULL,
    day date NOT NULL,
    tokens bigint NOT NULL DEFAULT 0,
    turns integer NOT NULL DEFAULT 0,
    PRIMARY KEY (scope, subject, day)
);
CREATE INDEX learning_chat_usage_day ON learning_chat_usage(day);

-- +goose Down
DROP TABLE learning_chat_usage;
