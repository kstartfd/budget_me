-- +goose Up
-- Income vs expense as a finite set → Postgres ENUM (not text+CHECK). Both the
-- transactions and categories tables carry it; a transaction of a given
-- direction may only point at a category of the same direction.
CREATE TYPE tx_direction AS ENUM ('expense', 'income');

-- Transient DEFAULT backfills existing rows as 'expense' (they ARE expenses —
-- amount_minor stays a positive magnitude, never flipped), then it's dropped:
-- this project keeps no permanent DB defaults, the app supplies direction on
-- insert.
ALTER TABLE transactions ADD COLUMN direction tx_direction NOT NULL DEFAULT 'expense';
ALTER TABLE transactions ALTER COLUMN direction DROP DEFAULT;

ALTER TABLE categories ADD COLUMN direction tx_direction NOT NULL DEFAULT 'expense';
ALTER TABLE categories ALTER COLUMN direction DROP DEFAULT;

-- Shared income preset categories (user_id NULL, identified by slug; the display
-- name comes from the frontend i18n by slug, so name stays NULL). Same style as
-- 0004: fixed UUIDs (continuing the 0004 sequence), literal created_at, idempotent
-- via ON CONFLICT (slug) DO NOTHING. Colors mirror the frontend income mock.
INSERT INTO categories (id, user_id, slug, name, color, created_at, direction) VALUES
    ('00000000-0000-4000-8000-000000000007', NULL, 'salary',    NULL, '#35b98f', '2026-01-01 00:00:00+00', 'income'),
    ('00000000-0000-4000-8000-000000000008', NULL, 'dividends', NULL, '#46c4c0', '2026-01-01 00:00:00+00', 'income')
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DELETE FROM categories
WHERE user_id IS NULL
  AND slug IN ('salary', 'dividends');

ALTER TABLE transactions DROP COLUMN direction;
ALTER TABLE categories DROP COLUMN direction;
DROP TYPE tx_direction;
