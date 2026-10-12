-- +goose Up
-- Planned expenses for a month — a checklist the user confirms as the spend
-- happens (confirm creates a transaction on the app side and stamps confirmed_at
-- here, kept for history so future months can suggest recurring plans). `month`
-- is the "YYYY-MM" bucket key (validated by the handler); both FKs cascade.
CREATE TABLE plans (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id  uuid NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    month        text NOT NULL,
    title        text NOT NULL,
    amount_minor bigint NOT NULL,
    confirmed_at timestamptz,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);

CREATE INDEX plans_user_month ON plans (user_id, month);

-- +goose Down
DROP TABLE plans;
