-- name: InsertCategory :one
INSERT INTO categories (
    id, user_id, slug, name, color, created_at, direction
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: ListCategoriesForUser :many
SELECT * FROM categories
WHERE user_id IS NULL                 -- общие (предустановленные)
   OR user_id = sqlc.arg(user_id)     -- личные этого юзера
ORDER BY user_id NULLS FIRST, created_at;


-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: CategoryDirection :one
SELECT direction FROM categories WHERE id = $1;

-- name: CategoryNameExists :one
SELECT EXISTS (
    SELECT 1 FROM categories
    WHERE user_id = sqlc.arg(user_id)
      AND lower(name) = lower(sqlc.arg(name)::text)
) AS taken;

-- name: CategoryNameTakenByOther :one
SELECT EXISTS (
    SELECT 1 FROM categories
    WHERE user_id = sqlc.arg(user_id)
      AND lower(name) = lower(sqlc.arg(name)::text)
      AND id <> sqlc.arg(exclude_id)     -- игнорируем саму редактируемую строку
) AS taken;

-- name: UpdateCategory :one
UPDATE categories
SET name = sqlc.arg(name),
    color = sqlc.arg(color)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);