-- The first learning loop: knowledge items, evidence, item state, activities.
-- See docs/learning/VERTICAL_SLICE_V1.md. Every query that reads a learner's
-- rows is scoped by user_id; an id of another learner is simply not found.

-- name: UpsertKBItem :one
-- An item is made from the library the first time the learner shows something
-- about it; the no-op update makes the row come back on a repeat.
INSERT INTO learning_knowledge_items (id, language, key, kind, skill, cefr_level, unit, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (language, key) DO UPDATE SET key = learning_knowledge_items.key
RETURNING *;

-- name: GetKBItem :one
SELECT * FROM learning_knowledge_items WHERE id = $1;

-- name: GetKBItemByKey :one
SELECT * FROM learning_knowledge_items WHERE language = $1 AND key = $2;

-- name: InsertLearningEvidence :execrows
-- One attempt gives at most one row, so a repeated write changes nothing.
INSERT INTO learning_evidence (
    id, user_id, teacher_id, item_id, skill, source, practice_mode, ref_type, ref_id,
    mode, outcome, score, weight, assisted, attempt_no, hints_used, occurred_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
ON CONFLICT (ref_type, ref_id) DO NOTHING;

-- name: CountLearningPathAttemptsOfExercise :one
SELECT COUNT(*)::integer FROM learning_path_attempts WHERE exercise_id = $1 AND user_id = $2;

-- name: EnsureLearnerItemState :exec
-- The row is made before it is locked, so two first answers cannot both start from nothing.
INSERT INTO learner_item_state (teacher_id, user_id, item_id, state, updated_at)
VALUES ($1, $2, $3, 'unseen', $4)
ON CONFLICT (teacher_id, item_id) DO NOTHING;

-- name: LockLearnerItemState :one
SELECT * FROM learner_item_state WHERE teacher_id = $1 AND item_id = $2 AND user_id = $3 FOR UPDATE;

-- name: SaveLearnerItemState :exec
UPDATE learner_item_state SET
    state = $4, score = $5, weighted_sum = $6, weight_sum = $7, evidence_count = $8,
    days = $9, recall_successes = $10, streak = $11, streak_day = $12, stage = $13,
    lapses = $14, last_evidence_at = $15, last_success_at = $16, due_at = $17, updated_at = $18
WHERE teacher_id = $1 AND item_id = $2 AND user_id = $3;

-- name: GetLearnerItemState :one
SELECT * FROM learner_item_state WHERE teacher_id = $1 AND item_id = $2 AND user_id = $3;

-- name: CountLearnerItemStates :many
SELECT state, COUNT(*)::integer AS count
FROM learner_item_state
WHERE teacher_id = $1 AND user_id = $2 AND evidence_count > 0
GROUP BY state;

-- name: CountLearnerReviews :one
-- due: asked for now or earlier; overdue: a day or more late; upcoming: still to come.
SELECT
    COUNT(*) FILTER (WHERE due_at <= @now::timestamptz)::integer AS due,
    COUNT(*) FILTER (WHERE due_at <= @overdue_before::timestamptz)::integer AS overdue,
    COUNT(*) FILTER (WHERE due_at > @now::timestamptz)::integer AS upcoming,
    MIN(due_at) FILTER (WHERE due_at > @now::timestamptz)::timestamptz AS next_due
FROM learner_item_state
WHERE teacher_id = @teacher_id AND user_id = @user_id AND evidence_count > 0 AND due_at IS NOT NULL;

-- name: CountLearningEvidence :one
SELECT COUNT(*)::integer AS total,
       COUNT(*) FILTER (WHERE occurred_at >= @since::timestamptz)::integer AS recent
FROM learning_evidence
WHERE teacher_id = @teacher_id AND user_id = @user_id;

-- name: ListDueReviewItems :many
-- Items asked for now or earlier, most overdue first, that still have an
-- exercise to be answered: a history that was cleared leaves nothing to ask.
SELECT s.item_id, s.state, s.stage, s.due_at, i.key, i.skill, i.cefr_level
FROM learner_item_state s
JOIN learning_knowledge_items i ON i.id = s.item_id
WHERE s.teacher_id = $1 AND s.user_id = $2 AND s.evidence_count > 0
  AND s.due_at IS NOT NULL AND s.due_at <= $3
  AND EXISTS (
      SELECT 1
      FROM learning_path_topics t
      JOIN learning_paths p ON p.id = t.path_id
      JOIN learning_path_exercises e ON e.topic_id = t.id
      JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
      WHERE p.user_id = $2 AND p.language = i.language AND t.source_id = i.key
        AND v.generation = (
            SELECT MAX(current.generation)
            FROM learning_path_exercise_versions current
            WHERE current.topic_id = e.topic_id
        )
  )
ORDER BY s.due_at, s.item_id
LIMIT $4;

-- name: PickReviewExercise :one
-- The exercise to ask for a due item: a produced answer rather than a pick,
-- and the one answered longest ago (never answered first).
SELECT e.*
FROM learning_path_exercises e
JOIN learning_path_topics t ON t.id = e.topic_id
JOIN learning_paths p ON p.id = t.path_id
JOIN learning_path_exercise_versions v ON v.exercise_id = e.id
WHERE p.user_id = $1 AND p.language = $2 AND t.source_id = $3
  AND v.generation = (
      SELECT MAX(current.generation)
      FROM learning_path_exercise_versions current
      WHERE current.topic_id = e.topic_id
  )
ORDER BY
    (e.kind IN ('choice', 'listen')),
    (SELECT MAX(a.created_at) FROM learning_path_attempts a WHERE a.exercise_id = e.id AND a.user_id = $1) NULLS FIRST,
    e.id
LIMIT 1;

-- name: GetCurrentPracticeTopic :one
-- The topic the learner works on next in the running program of a language.
SELECT t.*
FROM learning_path_topics t
JOIN learning_paths p ON p.id = t.path_id
WHERE p.user_id = $1 AND p.language = $2 AND p.status = 'active' AND t.status <> 'completed'
ORDER BY t.position, t.id
LIMIT 1;

-- name: GetLearningPathTopicByExercise :one
SELECT t.*
FROM learning_path_topics t
JOIN learning_path_exercises e ON e.topic_id = t.id
JOIN learning_paths p ON p.id = t.path_id
WHERE e.id = $1 AND p.user_id = $2;

-- name: InsertLearningActivity :execrows
-- A second open activity for the same item or topic is refused by the partial
-- unique indexes, which makes planning safe to repeat.
INSERT INTO learning_activities (id, user_id, teacher_id, type, status, reason, goal, item_id, topic_id, params, created_at)
VALUES ($1, $2, $3, $4, 'open', $5, $6, $7, $8, $9, $10)
ON CONFLICT DO NOTHING;

-- name: GetLearningActivity :one
SELECT * FROM learning_activities WHERE id = $1 AND user_id = $2;

-- name: ListOpenLearningActivities :many
SELECT * FROM learning_activities
WHERE teacher_id = $1 AND user_id = $2 AND status = 'open'
ORDER BY (type = 'review') DESC, created_at, id;

-- name: CompleteLearningActivity :execrows
UPDATE learning_activities SET status = 'completed', completed_at = $3
WHERE id = $1 AND user_id = $2 AND status = 'open';

-- name: CloseFinishedPracticeActivities :execrows
-- A practice activity is done when its topic is; the topic's own time is kept.
UPDATE learning_activities a
SET status = 'completed', completed_at = COALESCE(t.completed_at, $3)
FROM learning_path_topics t
WHERE a.teacher_id = $1 AND a.user_id = $2 AND a.type = 'practice' AND a.status = 'open'
  AND a.topic_id = t.id AND t.status = 'completed';

-- name: SkipStalePracticeActivities :execrows
-- The program was closed with the topic unfinished: the activity is no longer to be done.
UPDATE learning_activities a
SET status = 'skipped', completed_at = $3
FROM learning_path_topics t
JOIN learning_paths p ON p.id = t.path_id
WHERE a.teacher_id = $1 AND a.user_id = $2 AND a.type = 'practice' AND a.status = 'open'
  AND a.topic_id = t.id AND p.status <> 'active';

-- name: SkipStaleReviewActivities :execrows
-- The item was answered on another way and is not due any more.
UPDATE learning_activities a
SET status = 'skipped', completed_at = $3
FROM learner_item_state s
WHERE a.teacher_id = $1 AND a.user_id = $2 AND a.type = 'review' AND a.status = 'open'
  AND s.teacher_id = a.teacher_id AND s.item_id = a.item_id
  AND (s.due_at IS NULL OR s.due_at > $3);
