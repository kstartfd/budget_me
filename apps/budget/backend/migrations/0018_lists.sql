-- +goose Up
-- Named lists of small things to track and rate — the medicine cabinet, the
-- cosmetics shelf, and whatever else. Mirrors categories: a dictionary of
-- lists (shared presets via user_id NULL + identified by slug, or personal
-- via name) that items then belong to, same as categories dictionary
-- transactions.
CREATE TABLE item_lists (
    id         uuid PRIMARY KEY,
    user_id    uuid REFERENCES users(id) ON DELETE CASCADE,
    slug       text UNIQUE,
    name       text,
    color      text NOT NULL,
    created_at timestamptz NOT NULL
);

-- Whether an item helped — a fixed, small set of outcomes, so an ENUM, not
-- text+CHECK (project rule). "unknown" is the default: most items start
-- untried, not "no".
CREATE TYPE item_verdict AS ENUM ('works', 'meh', 'no', 'unknown');

-- rating is 1..10, always owner-checked at the Go edge (like amount_minor > 0
-- elsewhere) — the project keeps the DB to structure only, no CHECKs.
CREATE TABLE items (
    id          uuid PRIMARY KEY,
    list_id     uuid NOT NULL REFERENCES item_lists(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       text NOT NULL,
    purpose     text,
    verdict     item_verdict NOT NULL DEFAULT 'unknown',
    rating      smallint,
    note        text,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);

CREATE INDEX items_list_user ON items (list_id, user_id);

-- Two shared starter lists, same idempotent seeding style as
-- 0004_seed_categories.sql (fixed ids, ON CONFLICT DO NOTHING). Display name
-- comes from frontend i18n by slug, like preset categories.
INSERT INTO item_lists (id, user_id, slug, name, color, created_at) VALUES
    ('00000000-0000-4000-9000-000000000001', NULL, 'medications', NULL, '#ea6a6a', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-9000-000000000002', NULL, 'cosmetics',   NULL, '#e07ea8', '2026-01-01 00:00:00+00')
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DROP TABLE items;
DROP TABLE item_lists;
DROP TYPE item_verdict;
