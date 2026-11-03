-- +goose Up
-- Audio generated for an entry is cached globally by language and normalized
-- text. The binary lives on the server filesystem; this table is its index.
CREATE TABLE learning_audio_cache (
    id uuid PRIMARY KEY,
    language text NOT NULL,
    text text NOT NULL,
    text_hash bytea NOT NULL,
    content_hash bytea NOT NULL,
    storage_path text NOT NULL UNIQUE,
    content_type text NOT NULL,
    byte_size bigint NOT NULL,
    provider text NOT NULL,
    created_at timestamptz NOT NULL,
    last_used_at timestamptz NOT NULL,
    UNIQUE (language, text_hash)
);

-- +goose Down
DROP TABLE learning_audio_cache;
