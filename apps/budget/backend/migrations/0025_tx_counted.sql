-- +goose Up
-- Whether a transaction takes part in the money figures: the balance, "free to
-- spend" (income − spend) and the carry-over into the next period. A mandatory
-- payment can be left out of them while statistics (categories, trends, the
-- list, advice) still see every row. Only an expense may be left out; the
-- handler keeps income counted.
--
-- Transient DEFAULT backfills existing rows as counted (that is how they have
-- always behaved), then it's dropped: no permanent DB defaults, the app
-- supplies the flag on insert.
ALTER TABLE transactions ADD COLUMN counted boolean NOT NULL DEFAULT true;
ALTER TABLE transactions ALTER COLUMN counted DROP DEFAULT;

-- +goose Down
ALTER TABLE transactions DROP COLUMN counted;
