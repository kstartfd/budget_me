-- +goose Up
-- Changing the day a billing period opens re-cuts every period the app has ever
-- shown, so the figures carried across that boundary stop meaning anything. The
-- user confirms a reset instead: this column is the moment they did.
--
-- Nothing is deleted. Transactions before the cutoff stay fully visible in
-- history and analytics — they just stop feeding the running totals, so the
-- first period after the change starts clean rather than inheriting a carry-over
-- computed under the old period boundaries.
--
-- NULL = never reset (the default, and the only state before this migration).
ALTER TABLE users ADD COLUMN history_reset_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN history_reset_at;
