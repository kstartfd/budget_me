-- +goose Up
-- The lesson timer belongs to the learner, not to a topic: one sitting runs
-- while they move between topics. New lessons have no topic; the ones held
-- before keep the topic they were on.
ALTER TABLE learning_path_lessons ALTER COLUMN topic_id DROP NOT NULL;

-- +goose Down
DELETE FROM learning_path_lessons WHERE topic_id IS NULL;
ALTER TABLE learning_path_lessons ALTER COLUMN topic_id SET NOT NULL;
