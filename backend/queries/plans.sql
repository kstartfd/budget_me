-- name: ListPlansForMonth :many
SELECT * FROM plans
WHERE user_id = sqlc.arg(user_id) AND month = sqlc.arg(month)
ORDER BY confirmed_at NULLS FIRST, created_at;

-- name: InsertPlan :one
INSERT INTO plans (
    id, user_id, category_id, month, title, amount_minor, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(category_id), sqlc.arg(month),
    sqlc.arg(title), sqlc.arg(amount_minor), sqlc.arg(now), sqlc.arg(now)
)
RETURNING *;

-- name: UpdatePlan :one
UPDATE plans
SET category_id = sqlc.arg(category_id),
    title = sqlc.arg(title),
    amount_minor = sqlc.arg(amount_minor),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: ConfirmPlan :one
UPDATE plans
SET confirmed_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeletePlan :exec
DELETE FROM plans WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListPlanSuggestionsForMonth :many
WITH history AS (
    SELECT past.category_id,
           lower(btrim(past.title)) AS key,
           past.title,
           past.amount_minor,
           count(*) OVER (PARTITION BY past.category_id, lower(btrim(past.title))) AS seen_count,
           row_number() OVER (
               PARTITION BY past.category_id, lower(btrim(past.title))
               ORDER BY past.month DESC
           ) AS recency
    FROM plans past
    WHERE past.user_id = sqlc.arg(user_id)
      AND past.confirmed_at IS NOT NULL
      AND past.month >= sqlc.arg(from_month)
      AND past.month < sqlc.arg(target_month)
)
SELECT h.category_id, h.title, h.amount_minor, h.seen_count
FROM history h
WHERE h.recency = 1
  AND NOT EXISTS (
      SELECT 1 FROM plans taken
      WHERE taken.user_id = sqlc.arg(user_id)
        AND taken.month = sqlc.arg(target_month)
        AND taken.category_id = h.category_id
        AND lower(btrim(taken.title)) = h.key
  )
ORDER BY h.seen_count DESC, h.amount_minor DESC
LIMIT 12;
