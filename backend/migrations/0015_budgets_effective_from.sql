-- +goose Up
-- A limit stops being a per-month chore. Until now a row was "the limit FOR
-- 2026-07", so every new month started with none and had to be typed again.
-- `month` becomes `valid_from` — the first month the limit applies to — and the
-- limit in force for month X is the newest row with valid_from <= X. Setting a
-- limit once therefore carries it into every later month until it is changed.
--
-- Existing rows keep their meaning (a limit entered for 2026-07 still IS July's)
-- and simply stop vanishing at the month boundary. Months that were left blank
-- after a limit was set now inherit it — that is the point of the change.
ALTER TABLE budgets RENAME COLUMN month TO valid_from;

ALTER TABLE budgets DROP CONSTRAINT budgets_user_id_category_id_month_key;
ALTER TABLE budgets ADD CONSTRAINT budgets_user_id_category_id_valid_from_key
    UNIQUE (user_id, category_id, valid_from);

-- NULL limit = tombstone: "from this month on, this category has no limit".
-- Removing a limit cannot just delete the row — the previous row would then be
-- the newest one with valid_from <= X and the old limit would come straight
-- back. The tombstone ends the run while leaving history intact.
ALTER TABLE budgets ALTER COLUMN limit_minor DROP NOT NULL;

-- +goose Down
DELETE FROM budgets WHERE limit_minor IS NULL;

ALTER TABLE budgets ALTER COLUMN limit_minor SET NOT NULL;

ALTER TABLE budgets DROP CONSTRAINT budgets_user_id_category_id_valid_from_key;
ALTER TABLE budgets ADD CONSTRAINT budgets_user_id_category_id_month_key
    UNIQUE (user_id, category_id, valid_from);

ALTER TABLE budgets RENAME COLUMN valid_from TO month;
