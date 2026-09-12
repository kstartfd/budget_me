-- +goose Up
-- Day checklist. Nothing is ever carried over by writing rows — a day is a
-- QUERY: everything still open (done_on IS NULL) at or before that day, plus
-- whatever was ticked ON that day. So an untouched task simply reappears
-- tomorrow wearing the day it was originally planned for, with no nightly job
-- and no row ever rewritten at midnight — "висит с 9 сент." falls straight out
-- of `day - planned_for`.
--
-- planned_for / done_on are LOCAL calendar dates (plain `date`, no time, no
-- zone): the backend is UTC-only and does not know the user's zone (same
-- reasoning as settings.today), and the client is the one that knows which
-- wall-clock day a checkbox was ticked on.
CREATE TABLE tasks (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       text NOT NULL,
    note        text,
    planned_for date NOT NULL,
    done_on     date,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);

-- Partial index matched to the one hot read: a user's still-open tasks.
CREATE INDEX tasks_user_open ON tasks (user_id, planned_for) WHERE done_on IS NULL;
-- Backs the "done on day X" half of the same query.
CREATE INDEX tasks_user_done ON tasks (user_id, done_on) WHERE done_on IS NOT NULL;

-- +goose Down
DROP TABLE tasks;
