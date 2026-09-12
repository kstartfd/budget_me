-- name: ListVisibleToUser :one
SELECT EXISTS (
    SELECT 1 FROM item_lists
    WHERE id = sqlc.arg(list_id)
      AND (user_id IS NULL OR user_id = sqlc.arg(user_id))   -- общий ИЛИ свой
) AS visible;

-- name: InsertItem :one
INSERT INTO items (
    id, list_id, user_id, title, purpose, verdict, rating, note, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $9
)
RETURNING *;

-- name: ListItemsForList :many
SELECT * FROM items
WHERE list_id = sqlc.arg(list_id) AND user_id = sqlc.arg(user_id)
ORDER BY created_at;

-- name: UpdateItem :one
UPDATE items
SET title = sqlc.arg(title),
    purpose = sqlc.narg(purpose),
    verdict = sqlc.arg(verdict),
    rating = sqlc.narg(rating),
    note = sqlc.narg(note),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteItem :execrows
DELETE FROM items
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);
