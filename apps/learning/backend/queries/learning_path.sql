-- name: InsertLearningPathTest :one
INSERT INTO learning_path_tests (id, user_id, path_id, kind, language, translation_language, level, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'in_progress', $8) RETURNING *;

-- name: GetLearningPathTest :one
SELECT * FROM learning_path_tests WHERE id = $1 AND user_id = $2;

-- name: GetOpenLearningPathTest :one
SELECT * FROM learning_path_tests
WHERE user_id = $1 AND language = $2 AND kind = $3 AND status = 'in_progress'
ORDER BY created_at DESC, id LIMIT 1;

-- name: FinishLearningPathTest :exec
UPDATE learning_path_tests SET status = 'done', level_reached = $3, passed = $4, completed_at = $5
WHERE id = $1 AND user_id = $2 AND status = 'in_progress';

-- name: AdvanceLearningPathTest :exec
UPDATE learning_path_tests SET level = $3
WHERE id = $1 AND user_id = $2 AND status = 'in_progress';

-- name: InsertLearningPathTestAnswer :execrows
INSERT INTO learning_path_test_answers (test_id, question_key, answer, correct)
VALUES ($1, $2, $3, $4)
ON CONFLICT (test_id, question_key) DO NOTHING;

-- name: ListLearningPathTestAnswers :many
SELECT * FROM learning_path_test_answers WHERE test_id = $1;

-- name: InsertLearningPath :one
INSERT INTO learning_paths (id, user_id, language, translation_language, level, summary, status, source, placement_test_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, $9) RETURNING *;

-- name: GetActiveLearningPath :one
SELECT * FROM learning_paths WHERE user_id = $1 AND language = $2 AND status = 'active';

-- name: GetLearningPath :one
SELECT * FROM learning_paths WHERE id = $1 AND user_id = $2;

-- name: ArchiveLearningPath :execrows
UPDATE learning_paths SET status = $3, archived_at = $4
WHERE id = $1 AND user_id = $2 AND status = 'active';

-- name: ListLearningPathHistory :many
SELECT * FROM learning_paths
WHERE user_id = $1 AND status <> 'active'
ORDER BY created_at DESC, id DESC;

-- name: InsertLearningPathTopic :one
INSERT INTO learning_path_topics (id, path_id, position, skill, title, goal, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 'not_started', $7) RETURNING *;

-- name: ListLearningPathTopics :many
SELECT * FROM learning_path_topics WHERE path_id = $1 ORDER BY position, id;

-- name: GetLearningPathTopic :one
SELECT t.* FROM learning_path_topics t
JOIN learning_paths p ON p.id = t.path_id
WHERE t.id = $1 AND p.user_id = $2;

-- name: UpdateLearningPathTopicStatus :execrows
UPDATE learning_path_topics SET status = $3, completed_at = $4 WHERE id = $1 AND path_id = $2;

-- name: StartLearningPathTopic :execrows
UPDATE learning_path_topics SET status = 'in_progress', completed_at = NULL
WHERE id = $1 AND path_id = $2;

-- name: CompleteLearningPathTopicIfExercisesPassed :execrows
UPDATE learning_path_topics t
SET status = 'completed', completed_at = $3
WHERE t.id = $1
  AND t.status = 'in_progress'
  AND EXISTS (
      SELECT 1 FROM learning_paths p
      WHERE p.id = t.path_id AND p.status = 'active'
  )
  AND EXISTS (
      SELECT 1 FROM learning_path_exercises e
      JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
      WHERE e.topic_id = t.id
        AND v.generation = (
            SELECT MAX(current.generation)
            FROM learning_path_exercise_versions current
            WHERE current.topic_id = e.topic_id
        )
  )
  AND NOT EXISTS (
      SELECT 1 FROM learning_path_exercises e
      JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
      WHERE e.topic_id = t.id
        AND v.generation = (
            SELECT MAX(current.generation)
            FROM learning_path_exercise_versions current
            WHERE current.topic_id = e.topic_id
        )
        AND NOT EXISTS (
            SELECT 1 FROM learning_path_attempts a
            WHERE a.exercise_id = e.id AND a.user_id = $2 AND a.correct
        )
  );

-- name: InsertLearningPathExercise :one
INSERT INTO learning_path_exercises (id, topic_id, position, kind, text, prompt, options, answer, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: ListLearningPathExercises :many
SELECT e.* FROM learning_path_exercises e
JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
WHERE e.topic_id = $1
  AND v.generation = (
      SELECT MAX(current.generation)
      FROM learning_path_exercise_versions current
      WHERE current.topic_id = e.topic_id
  )
ORDER BY e.position, e.id;

-- name: GetNextLearningPathExerciseGeneration :one
SELECT COALESCE(MAX(generation), 0)::integer + 1
FROM learning_path_exercise_versions WHERE topic_id = $1;

-- name: LockLearningPathTopic :one
SELECT id FROM learning_path_topics WHERE id = $1 FOR UPDATE;

-- name: InsertLearningPathExerciseVersion :exec
INSERT INTO learning_path_exercise_versions (exercise_id, topic_id, generation)
VALUES ($1, $2, $3);

-- name: DeleteLearningPathExercises :exec
DELETE FROM learning_path_exercises WHERE topic_id = $1;

-- name: GetLearningPathExercise :one
SELECT e.* FROM learning_path_exercises e
JOIN learning_path_topics t ON t.id = e.topic_id
JOIN learning_paths p ON p.id = t.path_id
JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
WHERE e.id = $1 AND p.user_id = $2
  AND v.generation = (
      SELECT MAX(current.generation)
      FROM learning_path_exercise_versions current
      WHERE current.topic_id = e.topic_id
  );

-- name: InsertLearningPathAttempt :one
INSERT INTO learning_path_attempts (id, exercise_id, user_id, answer, correct, feedback, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: ListLearningPathAttemptsByTopic :many
SELECT a.* FROM learning_path_attempts a
JOIN learning_path_exercises e ON e.id = a.exercise_id
WHERE e.topic_id = $1 AND a.user_id = $2
ORDER BY a.created_at, a.id;
