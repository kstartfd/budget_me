-- +goose Up
-- Keep the existing relation and every foreign key intact. PostgreSQL renames
-- the table without copying rows; a simple updatable view preserves the old
-- application's kb_items reads and INSERT ... ON CONFLICT writes during a
-- blue-green deploy. The view intentionally exposes only the legacy columns.
ALTER TABLE kb_items RENAME TO learning_knowledge_items;

CREATE VIEW kb_items AS
SELECT id, language, key, kind, skill, cefr_level, unit, created_at
FROM learning_knowledge_items;

-- +goose Down
-- Roll back the application before removing its new table name. No rows are
-- copied or discarded: the same physical relation is renamed back.
DROP VIEW kb_items;
ALTER TABLE learning_knowledge_items RENAME TO kb_items;
