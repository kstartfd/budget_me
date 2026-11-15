-- +goose Up
-- Daily sets: a topic's own plan of "N new cards a day", with a weekly and a
-- monthly review of everything introduced in the calendar week / month.
--
-- learning_topic_plans: the learner's choices (cards a day, which side is shown
-- first) and the quota that accrues day by day. A day not opened adds to `credit`
-- instead of being lost, so the next set is bigger; `credited_on` is the last
-- local date it accrued for.
CREATE TABLE learning_topic_plans (
    topic_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    daily_new smallint NOT NULL DEFAULT 10 CHECK (daily_new BETWEEN 1 AND 50),
    front text NOT NULL DEFAULT 'translation' CHECK (front IN ('translation', 'text')),
    credit integer NOT NULL DEFAULT 0,
    credited_on date NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (topic_id, user_id) REFERENCES learning_topics(id, user_id) ON DELETE CASCADE
);

-- introduced_on: the local date an entry joined a daily set (NULL: not yet).
-- weak: the learner did not know it last time, so it comes back with the next set.
-- week_reviewed_on / month_reviewed_on: when it was last gone through in a weekly
-- / monthly review, so a long review can be done in parts and resumed.
ALTER TABLE learning_entries
    ADD COLUMN introduced_on date,
    ADD COLUMN weak boolean NOT NULL DEFAULT false,
    ADD COLUMN week_reviewed_on date,
    ADD COLUMN month_reviewed_on date;

CREATE INDEX learning_entries_introduced ON learning_entries(user_id, topic_id, introduced_on)
    WHERE introduced_on IS NOT NULL;

-- +goose Down
DROP INDEX learning_entries_introduced;
ALTER TABLE learning_entries
    DROP COLUMN month_reviewed_on,
    DROP COLUMN week_reviewed_on,
    DROP COLUMN weak,
    DROP COLUMN introduced_on;
DROP TABLE learning_topic_plans;
