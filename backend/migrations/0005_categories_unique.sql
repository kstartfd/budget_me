-- +goose Up
-- Drop duplicate personal categories, keeping the earliest per (user_id, name),
-- then enforce uniqueness so re-creating the same name can't duplicate again.
-- Presets (user_id IS NULL, identified by slug) are untouched.
DELETE FROM categories a
USING categories b
WHERE a.user_id IS NOT NULL
  AND b.user_id = a.user_id
  AND lower(b.name) = lower(a.name)
  AND (b.created_at < a.created_at
       OR (b.created_at = a.created_at AND b.id < a.id));

CREATE UNIQUE INDEX categories_user_name_uniq
    ON categories (user_id, lower(name))
    WHERE user_id IS NOT NULL;

-- +goose Down
DROP INDEX categories_user_name_uniq;
