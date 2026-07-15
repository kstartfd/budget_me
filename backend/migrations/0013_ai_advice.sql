-- +goose Up
-- Cached LLM advice, one row per (user, month). The model takes 3–18s to answer,
-- so it is NEVER called inside a request: the API serves whatever is cached here
-- and regenerates in the background when the numbers move. `facts_hash` is what
-- the advice was written from — when this month's facts hash to something else,
-- the text is stale and worth rewriting.
CREATE TABLE ai_advice (
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    month       text        NOT NULL,
    facts_hash  text        NOT NULL,
    text        text        NOT NULL,
    created_at  timestamptz NOT NULL,
    PRIMARY KEY (user_id, month)
);

-- +goose Down
DROP TABLE ai_advice;
