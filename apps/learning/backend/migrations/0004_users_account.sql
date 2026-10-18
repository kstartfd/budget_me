-- +goose Up
-- Identity moved to the shared account database. account_id points at the
-- person's account there (another database, so no foreign key); the other
-- columns stay as this app's copy of the profile.
ALTER TABLE users ADD COLUMN account_id uuid UNIQUE;

-- +goose Down
ALTER TABLE users DROP COLUMN account_id;
