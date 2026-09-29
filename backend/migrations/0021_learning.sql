-- +goose Up
CREATE TYPE learning_entry_kind AS ENUM ('word', 'phrase');
CREATE TYPE learning_entry_status AS ENUM ('inbox', 'learning', 'known');
CREATE TYPE learning_token_scope AS ENUM ('capture', 'read_write');
CREATE TYPE learning_capture_status AS ENUM ('pending', 'processing', 'done', 'failed');

CREATE TABLE learning_topics (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL,
    language text NOT NULL,
    translation_language text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (id, user_id)
);
CREATE INDEX learning_topics_user ON learning_topics(user_id, created_at);

CREATE TABLE learning_entries (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id uuid NOT NULL,
    kind learning_entry_kind NOT NULL,
    text text NOT NULL,
    transcription text NOT NULL,
    translation text NOT NULL,
    native_usage text NOT NULL,
    example text NOT NULL,
    rule text NOT NULL,
    status learning_entry_status NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (topic_id, user_id) REFERENCES learning_topics(id, user_id) ON DELETE CASCADE
);
CREATE INDEX learning_entries_topic ON learning_entries(user_id, topic_id, created_at DESC, id);

CREATE TABLE learning_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL,
    prefix text NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    scope learning_token_scope NOT NULL,
    created_at timestamptz NOT NULL,
    last_used_at timestamptz,
    revoked_at timestamptz
);
CREATE INDEX learning_tokens_user ON learning_tokens(user_id, created_at);

CREATE TABLE learning_captures (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_key text NOT NULL,
    payload jsonb NOT NULL,
    status learning_capture_status NOT NULL,
    attempts integer NOT NULL,
    available_at timestamptz NOT NULL,
    locked_until timestamptz,
    lease_id uuid,
    entry_id uuid,
    error text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (user_id, request_key)
);
CREATE INDEX learning_captures_ready ON learning_captures(available_at, created_at)
    WHERE status IN ('pending', 'processing');

-- +goose Down
DROP TABLE learning_captures;
DROP TABLE learning_tokens;
DROP TABLE learning_entries;
DROP TABLE learning_topics;
DROP TYPE learning_capture_status;
DROP TYPE learning_token_scope;
DROP TYPE learning_entry_status;
DROP TYPE learning_entry_kind;
