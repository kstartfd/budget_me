-- +goose Up
-- Deferred account deletion: the request is a timestamp on the user row, not an
-- immediate DROP. A worker (cmd/worker, run by cron) removes the rows once the
-- grace period is up, so the user can change their mind by signing back in and
-- cancelling — until then the account keeps working. NULL = no request pending.
-- The data itself goes with the user: every table already cascades on users(id).
ALTER TABLE users ADD COLUMN deletion_requested_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN deletion_requested_at;
