-- +goose Up
-- Per-category monthly spending limits. `month` is the "YYYY-MM" bucket key the
-- frontend uses (not a finite set, so plain text; the handler validates the
-- format). One limit per (user, category, month) — the UNIQUE key backs the
-- upsert. Both FKs cascade: a deleted user or category takes its limits with it.
CREATE TABLE budgets (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id  uuid NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    month        text NOT NULL,
    limit_minor  bigint NOT NULL,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    UNIQUE (user_id, category_id, month)
);

-- +goose Down
DROP TABLE budgets;
