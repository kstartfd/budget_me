-- +goose Up
-- Shared preset categories (user_id NULL, identified by slug; the display name
-- comes from the frontend i18n by slug, so name stays NULL). Idempotent via
-- ON CONFLICT (slug) DO NOTHING — safe to re-run on every deploy. Fixed UUIDs
-- keep preset ids stable across environments; created_at is a literal (the DB
-- holds no now()/defaults).
INSERT INTO categories (id, user_id, slug, name, color, created_at) VALUES
    ('00000000-0000-4000-8000-000000000001', NULL, 'coffee',    NULL, '#e0a33e', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-8000-000000000002', NULL, 'groceries', NULL, '#35b98f', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-8000-000000000003', NULL, 'dining',    NULL, '#9b8cff', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-8000-000000000004', NULL, 'transport', NULL, '#5b8def', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-8000-000000000005', NULL, 'fun',       NULL, '#e07ea8', '2026-01-01 00:00:00+00'),
    ('00000000-0000-4000-8000-000000000006', NULL, 'subs',      NULL, '#46c4c0', '2026-01-01 00:00:00+00')
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DELETE FROM categories
WHERE user_id IS NULL
  AND slug IN ('coffee', 'groceries', 'dining', 'transport', 'fun', 'subs');
