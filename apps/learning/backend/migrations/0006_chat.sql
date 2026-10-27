-- +goose Up
CREATE TYPE learning_chat_role AS ENUM ('user', 'assistant');

-- One chat per entry, created with the first message. The full history stays in
-- learning_chat_messages; `summary` only shortens what is sent to the model:
-- it stands in for every message up to `summary_upto`.
CREATE TABLE learning_chats (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id uuid NOT NULL UNIQUE REFERENCES learning_entries(id) ON DELETE CASCADE,
    summary text NOT NULL DEFAULT '',
    summary_upto bigint NOT NULL DEFAULT 0,
    -- Tokens the provider counted for the last exchange (prompt + reply): the
    -- size the next request starts from, and the trigger for compaction.
    context_tokens integer NOT NULL DEFAULT 0,
    -- Held while an answer or a compaction is in flight, so two of them never
    -- interleave in one chat. It expires on its own if the server dies.
    busy_until timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX learning_chats_user ON learning_chats(user_id);

CREATE TABLE learning_chat_messages (
    id bigserial PRIMARY KEY,
    chat_id uuid NOT NULL REFERENCES learning_chats(id) ON DELETE CASCADE,
    role learning_chat_role NOT NULL,
    content text NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX learning_chat_messages_chat ON learning_chat_messages(chat_id, id);

-- +goose Down
DROP TABLE learning_chat_messages;
DROP TABLE learning_chats;
DROP TYPE learning_chat_role;
