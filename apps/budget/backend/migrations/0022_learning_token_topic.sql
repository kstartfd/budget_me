-- +goose Up
ALTER TABLE learning_tokens ADD COLUMN topic_id uuid REFERENCES learning_topics(id) ON DELETE CASCADE;
CREATE INDEX learning_tokens_topic ON learning_tokens(topic_id);

-- +goose Down
DROP INDEX learning_tokens_topic;
ALTER TABLE learning_tokens DROP COLUMN topic_id;
