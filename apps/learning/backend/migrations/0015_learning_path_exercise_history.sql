-- +goose Up
-- Keep each generated exercise set and its attempts when a topic is reset.
CREATE TABLE learning_path_exercise_versions (
    exercise_id uuid PRIMARY KEY REFERENCES learning_path_exercises(id) ON DELETE CASCADE,
    topic_id uuid NOT NULL REFERENCES learning_path_topics(id) ON DELETE CASCADE,
    generation integer NOT NULL,
    UNIQUE (topic_id, generation, exercise_id)
);

CREATE INDEX learning_path_exercise_versions_topic
    ON learning_path_exercise_versions(topic_id, generation, exercise_id);

INSERT INTO learning_path_exercise_versions (exercise_id, topic_id, generation)
SELECT id, topic_id, 1 FROM learning_path_exercises;

-- +goose Down
DROP TABLE learning_path_exercise_versions;
