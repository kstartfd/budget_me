-- name: InsertTransaction :one
INSERT INTO transactions (
    id, user_id, category_id, amount_minor, title, spent_at, created_at, direction, counted
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListTransactionsInRange :many
SELECT * FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
ORDER BY spent_at DESC;

-- name: SumSpentInRange :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS spent_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND direction = 'expense';

-- name: SumIncomeInRange :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS income_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND direction = 'income';

-- name: SumCountedSpentInRange :one
-- Расход за период, который участвует в РАСЧЁТЕ («Свободно», полоска, перенос):
-- без трат с выключенной галочкой counted. Статистика (SumSpentInRange и
-- остальные Sum*) по-прежнему видит все траты.
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS spent_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND direction = 'expense'
  AND counted;

-- name: SumTotalsInRange :one
-- Расход и приход за период ОДНОЙ строкой, с необязательным сужением по категории
-- (category_id NULL → без фильтра). Питает шапку экрана «Транзакции»: сумма под
-- текущим фильтром считается на бэке, а не складывается на клиенте из выданного списка.
SELECT
    COALESCE(SUM(amount_minor) FILTER (WHERE direction = 'expense'), 0)::bigint AS spent_minor,
    COALESCE(SUM(amount_minor) FILTER (WHERE direction = 'income'),  0)::bigint AS income_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id));

-- name: SumSpentByCategoryInRange :many
SELECT
    category_id,
    COALESCE(SUM(amount_minor), 0)::bigint AS spent_minor
FROM transactions
WHERE user_id  =  sqlc.arg(user_id)
  AND spent_at >= sqlc.arg(from_at)
  AND spent_at <  sqlc.arg(to_at)
  AND direction = 'expense'
GROUP BY category_id
ORDER BY spent_minor DESC;

-- name: ListRecentTransactions :many
SELECT * FROM transactions
WHERE user_id = sqlc.arg(user_id)
ORDER BY spent_at DESC
LIMIT sqlc.arg(lim);

-- name: CategoryVisibleToUser :one
SELECT EXISTS (
    SELECT 1 FROM categories
    WHERE id = sqlc.arg(category_id)
      AND (user_id IS NULL OR user_id = sqlc.arg(user_id))   -- общая ИЛИ своя
) AS visible;

-- name: DeleteTransaction :execrows
DELETE FROM transactions
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);   -- ownership в самом WHERE

-- name: UpdateTransaction :one
UPDATE transactions
SET category_id = sqlc.arg(category_id),
    amount_minor = sqlc.arg(amount_minor),
    title = sqlc.arg(title),
    spent_at = sqlc.arg(spent_at),
    direction = sqlc.arg(direction),
    counted = sqlc.arg(counted)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)   -- created_at не трогаем
RETURNING *;

-- name: SumSpentByPeriod :many
-- Расход, свёрнутый по РАСЧЁТНЫМ периодам (месяц может начинаться не с 1-го).
-- Сдвигаем дату траты назад на (start_day - 1) дней и берём месяц результата:
-- при start_day=5 трата 3 июля уедет в июнь — ровно в тот период, что открылся
-- 5 июня. Ключ периода = месяц его НАЧАЛА, как и на фронте (periodOpenKey).
-- Пустые месяцы сюда не попадают (GROUP BY по факту) — нули дошьёт хендлер.
SELECT
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
GROUP BY period
ORDER BY period;

-- name: SumSignedSince :one
-- Net movement since the balance anchor: income adds, expense subtracts. Feeds
-- the running balance (anchor + this).
--
-- Keyed on created_at, NOT spent_at: the anchor is stamped with server time,
-- while spent_at carries the client's local date+time. Comparing the two would
-- let a spend entered BEFORE the balance was stated land after the anchor (or
-- vice versa) purely through a timezone offset, and get counted twice. "Every
-- entry made since you told me the balance" is unambiguous and needs no clock
-- agreement.
--
-- Rows with counted = false (e.g. a mandatory payment the user keeps out of the
-- calculation) do not move the balance.
SELECT COALESCE(SUM(
    CASE WHEN direction = 'income' THEN amount_minor ELSE -amount_minor END
), 0)::bigint AS net_minor
FROM transactions
WHERE user_id    =  sqlc.arg(user_id)
  AND created_at >= sqlc.arg(from_at)
  AND counted;
