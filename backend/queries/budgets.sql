-- name: ListBudgetsForMonth :many
SELECT * FROM budgets
WHERE user_id = sqlc.arg(user_id) AND month = sqlc.arg(month);

-- name: UpsertBudget :one
INSERT INTO budgets (
    id, user_id, category_id, month, limit_minor, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(category_id),
    sqlc.arg(month), sqlc.arg(limit_minor), sqlc.arg(now), sqlc.arg(now)
)
ON CONFLICT (user_id, category_id, month)
DO UPDATE SET limit_minor = EXCLUDED.limit_minor, updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteBudget :exec
DELETE FROM budgets
WHERE user_id = sqlc.arg(user_id)
  AND category_id = sqlc.arg(category_id)
  AND month = sqlc.arg(month);

-- name: SumSpentByPeriodAndCategory :many
-- Feeds the limit recommendations: spend per (billing period, category) over a
-- window of past periods. Same bucketing as the trend — shift spent_at back by
-- (month_start_day - 1) days, then truncate to a month.
SELECT
    category_id,
    to_char(
        date_trunc(
            'month',
            (spent_at AT TIME ZONE sqlc.arg(tz)::text) - make_interval(days => sqlc.arg(start_offset)::int)
        ),
        'YYYY-MM'
    ) AS period,
    COALESCE(SUM(amount_minor), 0)::bigint AS spent_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND direction = 'expense'
  AND category_id IS NOT NULL
GROUP BY category_id, period;
