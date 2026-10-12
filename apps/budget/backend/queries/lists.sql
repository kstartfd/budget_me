-- name: InsertList :one
INSERT INTO item_lists (
    id, user_id, name, color, created_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListListsForUser :many
SELECT * FROM item_lists
WHERE user_id IS NULL                 -- общие (предустановленные)
   OR user_id = sqlc.arg(user_id)     -- личные этого юзера
ORDER BY user_id NULLS FIRST, created_at;

-- name: DeleteList :execrows
DELETE FROM item_lists
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpdateList :one
-- Same guard as DeleteList: WHERE user_id = @user_id can never match a preset
-- row (user_id IS NULL there), so a preset can't be renamed this way either.
UPDATE item_lists
SET name = sqlc.arg(name), color = sqlc.arg(color)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;
