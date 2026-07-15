-- +goose Up
-- Finite currency set → Postgres ENUM (not text+CHECK).
CREATE TYPE currency_code AS ENUM ('BYN', 'USD', 'EUR', 'RUB');

-- Per-user settings live on the user row (1:1). Transient DEFAULT backfills
-- existing users, then it's dropped — the app supplies the value on insert
-- (this project keeps no permanent DB defaults). Default start day is the 1st.
ALTER TABLE users ADD COLUMN month_start_day smallint NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN currency currency_code NOT NULL DEFAULT 'BYN';
ALTER TABLE users ALTER COLUMN month_start_day DROP DEFAULT;
ALTER TABLE users ALTER COLUMN currency DROP DEFAULT;

-- +goose Down
ALTER TABLE users DROP COLUMN month_start_day;
ALTER TABLE users DROP COLUMN currency;
DROP TYPE currency_code;
