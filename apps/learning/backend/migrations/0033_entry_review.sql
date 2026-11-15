-- +goose Up
-- Spaced repetition of saved entries: a Leitner-style stage and the day the
-- entry is next due. An entry never reviewed (next_review_at NULL) is new and
-- due at once; the stage grows with each remembered review and falls back to 0
-- when forgotten. Reaching the last stage marks the entry known.
ALTER TABLE learning_entries
    ADD COLUMN review_stage smallint NOT NULL DEFAULT 0,
    ADD COLUMN next_review_at timestamptz,
    ADD COLUMN last_reviewed_at timestamptz;

CREATE INDEX learning_entries_due ON learning_entries(user_id, topic_id, next_review_at)
    WHERE status <> 'known';

-- +goose Down
DROP INDEX learning_entries_due;
ALTER TABLE learning_entries
    DROP COLUMN last_reviewed_at,
    DROP COLUMN next_review_at,
    DROP COLUMN review_stage;
