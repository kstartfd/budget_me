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

-- name: InsertLearningPathTestQuestion :exec
INSERT INTO learning_path_test_questions (test_id, question_key, level, position, skill, text, prompt, options, answer)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListLearningPathTestQuestions :many
SELECT * FROM learning_path_test_questions WHERE test_id = $1 ORDER BY level, position;

-- name: InsertLearningPath :one
INSERT INTO learning_paths (id, user_id, language, translation_language, level, summary, status, source, placement_test_id, preferences, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, $9, $10) RETURNING *;

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

-- name: DeleteLearningPathHistory :execrows
-- Closed programs of one language, with everything under them (topics, exercises,
-- attempts, lessons, topic chats, final tests: all go with the program).
DELETE FROM learning_paths WHERE user_id = $1 AND language = $2 AND status <> 'active';

-- name: InsertLearningPathTopic :one
INSERT INTO learning_path_topics (id, path_id, position, skill, title, goal, unit, theory, vocabulary, status, created_at, source_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'not_started', $10, $11) RETURNING *;

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
INSERT INTO learning_path_exercises (id, topic_id, position, kind, text, prompt, options, answer, rule, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *;

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
INSERT INTO learning_path_attempts (id, exercise_id, user_id, answer, correct, feedback, lesson_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: ListLearningPathAttemptsByTopic :many
SELECT a.* FROM learning_path_attempts a
JOIN learning_path_exercises e ON e.id = a.exercise_id
WHERE e.topic_id = $1 AND a.user_id = $2
ORDER BY a.created_at, a.id;

-- name: InsertLearningPathLesson :one
INSERT INTO learning_path_lessons (id, user_id, status, started_at, ends_at)
VALUES ($1, $2, 'active', $3, $4) RETURNING *;

-- name: GetActiveLearningPathLesson :one
SELECT * FROM learning_path_lessons WHERE user_id = $1 AND status = 'active';

-- name: GetLearningPathLesson :one
SELECT * FROM learning_path_lessons WHERE id = $1 AND user_id = $2;

-- name: ListLearningPathLessons :many
SELECT * FROM learning_path_lessons
WHERE user_id = $1
ORDER BY started_at DESC, id DESC
LIMIT 30;

-- Closing a lesson also closes a pause that was still open, so the paused
-- time is never lost.
-- name: FinishLearningPathLesson :one
UPDATE learning_path_lessons SET
    status = 'finished',
    finished_at = sqlc.arg(finished_at),
    paused_seconds = paused_seconds + CASE
        WHEN paused_at IS NULL THEN 0
        ELSE GREATEST(0, EXTRACT(EPOCH FROM (sqlc.arg(finished_at)::timestamptz - paused_at)))::integer END,
    paused_at = NULL
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND status = 'active'
RETURNING *;

-- name: PauseLearningPathLesson :one
UPDATE learning_path_lessons SET paused_at = $3, pause_count = pause_count + 1
WHERE id = $1 AND user_id = $2 AND status = 'active' AND paused_at IS NULL
RETURNING *;

-- name: ResumeLearningPathLesson :one
UPDATE learning_path_lessons SET
    paused_seconds = paused_seconds + GREATEST(0, EXTRACT(EPOCH FROM ($3::timestamptz - paused_at)))::integer,
    paused_at = NULL
WHERE id = $1 AND user_id = $2 AND status = 'active' AND paused_at IS NOT NULL
RETURNING *;

-- name: ListLearningPathLessonStats :many
SELECT lesson_id::uuid AS lesson_id,
    count(*)::integer AS attempts,
    (count(*) FILTER (WHERE correct))::integer AS correct
FROM learning_path_attempts
WHERE lesson_id = ANY(sqlc.arg(lesson_ids)::uuid[])
GROUP BY lesson_id;

-- name: ListLearningTeachers :many
SELECT * FROM learning_teachers WHERE user_id = $1 ORDER BY created_at, id;

-- name: GetLearningTeacherByLanguage :one
SELECT * FROM learning_teachers WHERE user_id = $1 AND language = $2;

-- name: InsertLearningTeacher :one
INSERT INTO learning_teachers (id, user_id, language, translation_language, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, language) DO UPDATE SET language = learning_teachers.language
RETURNING *;

-- name: GetLearningTeacher :one
SELECT * FROM learning_teachers WHERE id = $1 AND user_id = $2;

-- name: DeleteLearningTeacher :execrows
DELETE FROM learning_teachers WHERE id = $1 AND user_id = $2;

-- name: ListActiveLearningPaths :many
SELECT * FROM learning_paths WHERE user_id = $1 AND status = 'active';

-- name: CountLearningPathExercises :many
-- What each topic's current exercise set holds, and how many of its exercises the
-- learner has answered right. Only the current set counts: a regenerated topic
-- starts from nothing again, whatever was answered in the sets before.
SELECT e.topic_id,
       count(*)::int AS total,
       (count(*) FILTER (WHERE EXISTS (
           SELECT 1 FROM learning_path_attempts a
           WHERE a.exercise_id = e.id AND a.user_id = $2 AND a.correct
       )))::int AS done
FROM learning_path_exercises e
JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
JOIN learning_path_topics t ON t.id = e.topic_id
WHERE t.path_id = $1
  AND v.generation = (
      SELECT MAX(current.generation)
      FROM learning_path_exercise_versions current
      WHERE current.topic_id = e.topic_id
  )
GROUP BY e.topic_id;

-- name: CountLearningPathTopics :many
SELECT path_id, count(*)::int AS total, (count(*) FILTER (WHERE status = 'completed'))::int AS done
FROM learning_path_topics WHERE path_id = ANY($1::uuid[]) GROUP BY path_id;

-- name: InsertLearningPathJob :one
INSERT INTO learning_path_jobs (id, user_id, test_id, status, preferences, created_at, updated_at)
VALUES ($1, $2, $3, 'queued', $4, $5, $5) RETURNING *;

-- name: GetLearningPathJob :one
SELECT * FROM learning_path_jobs WHERE id = $1 AND user_id = $2;

-- name: GetLatestLearningPathJobByTest :one
SELECT * FROM learning_path_jobs WHERE test_id = $1 AND user_id = $2
ORDER BY created_at DESC, id LIMIT 1;

-- name: SetLearningPathJobRunning :exec
UPDATE learning_path_jobs SET status = 'running', updated_at = $2 WHERE id = $1;

-- name: FinishLearningPathJob :exec
UPDATE learning_path_jobs SET status = $2, path_id = $3, error = $4, updated_at = $5 WHERE id = $1;
