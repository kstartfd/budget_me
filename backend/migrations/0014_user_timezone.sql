-- +goose Up
-- The user's last known IANA zone. The request path does NOT read this — there
-- the client sends its zone with every call (see 0012), which is always current.
-- This column exists for work that runs with no client attached: cmd/worker
-- writes the nightly advice, and without a zone it could not say which instants
-- make up "this month" — it would bucket by UTC while the API buckets by the
-- user's calendar, the facts would hash differently, and every visit would pay
-- for a needless regeneration.
ALTER TABLE users ADD COLUMN timezone text NOT NULL DEFAULT 'Europe/Minsk';
ALTER TABLE users ALTER COLUMN timezone DROP DEFAULT;

-- +goose Down
ALTER TABLE users DROP COLUMN timezone;
