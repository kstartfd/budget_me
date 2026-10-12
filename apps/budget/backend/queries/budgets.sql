-- name: ListBudgetsEffectiveIn :many
-- The limit in force for `month`, one row per category: the newest run that has
-- already started (valid_from <= month). DISTINCT ON keeps the first row of each
-- category group, and the ORDER BY makes that the latest start — so a limit set
-- in March is still the answer in July unless something newer replaced it.
--
-- Tombstones (NULL limit_minor) come back too. They are the difference between
-- "never had a limit" and "the limit was removed", and only the caller can say
-- which of the two it wants; the handler drops them.
SELECT DISTINCT ON (category_id) *
FROM budgets
WHERE user_id = sqlc.arg(user_id)
  AND valid_from <= sqlc.arg(month)
ORDER BY category_id, valid_from DESC;

-- name: UpsertBudget :one
-- Writes the run that starts at `valid_from`. A second save in the same month
-- corrects that run rather than opening another (the UNIQUE key backs it), so
-- editing today's limit twice does not leave a stray row behind.
INSERT INTO budgets (
    id, user_id, category_id, valid_from, limit_minor, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(category_id),
    sqlc.arg(valid_from), sqlc.arg(limit_minor), sqlc.arg(now), sqlc.arg(now)
)
ON CONFLICT (user_id, category_id, valid_from)
DO UPDATE SET limit_minor = EXCLUDED.limit_minor, updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: HasBudgetBefore :one
-- Is there an older run to inherit from? Decides how a removal is recorded: with
-- history behind it the limit needs a tombstone, without it the rows can simply
-- go (see DeleteBudgetsFrom).
SELECT EXISTS (
    SELECT 1 FROM budgets
    WHERE user_id     = sqlc.arg(user_id)
      AND category_id = sqlc.arg(category_id)
      AND valid_from  < sqlc.arg(month)
);

-- name: DeleteBudgetsFrom :exec
-- Erases this month's row and anything later — used only when no earlier run
-- exists, where a tombstone would preserve nothing.
DELETE FROM budgets
WHERE user_id     = sqlc.arg(user_id)
  AND category_id = sqlc.arg(category_id)
  AND valid_from >= sqlc.arg(month);

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
