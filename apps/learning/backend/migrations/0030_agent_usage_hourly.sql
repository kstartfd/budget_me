-- +goose Up
-- The sums of learning_agent_usage are kept per hour, not per day: some providers
-- price by the time of day (DeepSeek halves its prices off-peak), and the hour is
-- what the cost is worked out from at read time. Rows from before this
-- migration keep their day, at its first hour.
ALTER TABLE learning_agent_usage ADD COLUMN hour timestamptz;
UPDATE learning_agent_usage SET hour = day::timestamp AT TIME ZONE 'UTC';
ALTER TABLE learning_agent_usage ALTER COLUMN hour SET NOT NULL;
ALTER TABLE learning_agent_usage DROP CONSTRAINT learning_agent_usage_pkey;
ALTER TABLE learning_agent_usage DROP COLUMN day;
ALTER TABLE learning_agent_usage ADD PRIMARY KEY (user_id, hour, provider, model, purpose);
CREATE INDEX learning_agent_usage_hour ON learning_agent_usage(hour);

-- +goose Down
CREATE TABLE learning_agent_usage_by_day AS
SELECT user_id, (hour AT TIME ZONE 'UTC')::date AS day, provider, model, purpose,
       sum(calls)::integer AS calls, sum(input_tokens) AS input_tokens,
       sum(cached_tokens) AS cached_tokens, sum(output_tokens) AS output_tokens
FROM learning_agent_usage
GROUP BY user_id, (hour AT TIME ZONE 'UTC')::date, provider, model, purpose;
DROP TABLE learning_agent_usage;
ALTER TABLE learning_agent_usage_by_day RENAME TO learning_agent_usage;
ALTER TABLE learning_agent_usage
    ALTER COLUMN user_id SET NOT NULL, ALTER COLUMN day SET NOT NULL, ALTER COLUMN provider SET NOT NULL,
    ALTER COLUMN model SET NOT NULL, ALTER COLUMN purpose SET NOT NULL,
    ALTER COLUMN calls SET NOT NULL, ALTER COLUMN calls SET DEFAULT 0,
    ALTER COLUMN input_tokens SET NOT NULL, ALTER COLUMN input_tokens SET DEFAULT 0,
    ALTER COLUMN cached_tokens SET NOT NULL, ALTER COLUMN cached_tokens SET DEFAULT 0,
    ALTER COLUMN output_tokens SET NOT NULL, ALTER COLUMN output_tokens SET DEFAULT 0,
    ADD PRIMARY KEY (user_id, day, provider, model, purpose),
    ADD FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX learning_agent_usage_day ON learning_agent_usage(day);
