-- +goose Up
-- One photo per item, stored on local disk (the VPS filesystem — no S3, see
-- internal/photostore) rather than as a DB blob. photo_path is the path
-- relative to the uploads directory (e.g. "items/<uuid>.jpg"); NULL means the
-- item has no photo. The file itself is already a small, fixed-size JPEG by
-- the time it lands here — photostore resizes and recompresses on upload.
ALTER TABLE items ADD COLUMN photo_path text;

-- +goose Down
ALTER TABLE items DROP COLUMN photo_path;
