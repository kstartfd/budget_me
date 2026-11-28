-- name: InsertAssessment :one
INSERT INTO learning_path_tests (
    id, user_id, kind, language, translation_language, level, status, created_at,
    assessment_profile_id, assessment_type, assessment_status, start_band,
    assessment_policy, assessment_explanation_language
) VALUES ($1, $2, 'placement', $3, $4, $5, 'in_progress', $6, $7, $8, 'created', $5, $9, $10)
RETURNING *;

-- name: GetOpenAssessment :one
SELECT * FROM learning_path_tests WHERE assessment_profile_id = $1 AND user_id = $2
    AND assessment_type = $3 AND status = 'in_progress';

-- name: LockAssessment :one
SELECT * FROM learning_path_tests WHERE id = $1 AND user_id = $2 AND assessment_type IS NOT NULL FOR UPDATE;

-- name: UpdateAssessmentState :one
UPDATE learning_path_tests SET assessment_status = $3, level = $4, assessment_batch = $5
WHERE id = $1 AND user_id = $2 AND status = 'in_progress' RETURNING *;

-- name: FinishAssessment :one
UPDATE learning_path_tests SET status = 'done', completed_at = $3, level_reached = $4,
    assessment_status = $5, assessment_stop_reason = $6, assessment_result = $7
WHERE id = $1 AND user_id = $2 AND status = 'in_progress' RETURNING *;

-- name: InsertAssessmentQuestion :exec
INSERT INTO learning_path_test_questions (
    test_id, question_key, level, position, skill, text, prompt, options, answer,
    assessment_item_id, assessment_item_key, assessment_category, assessment_objective, assessment_format,
    assessment_version, assessment_origin, assessment_fingerprint, assessment_batch, assessment_alternatives,
    assessment_repeated
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20);

-- name: PriorAssessmentPrompts :many
SELECT DISTINCT q.assessment_fingerprint FROM learning_path_test_questions q
JOIN learning_path_tests t ON t.id = q.test_id
WHERE t.assessment_profile_id = $1 AND t.user_id = $2 AND t.id <> $3 AND q.assessment_fingerprint IS NOT NULL;

-- name: InsertAssessmentAnswer :execrows
INSERT INTO learning_path_test_answers (
    test_id, question_key, answer, correct, assessment_attempt_id, assessment_outcome, assessment_assisted
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (test_id, question_key) DO NOTHING;

-- name: GetAssessmentSection :one
SELECT s.id, s.skill, s.position, s.status, s.policy_snapshot, s.blueprint_snapshot,
       s.current_band, s.current_batch, s.answered, s.stop_reason, s.result_snapshot
FROM learning_assessment_sections s
JOIN learning_path_tests t ON t.id = s.test_id
WHERE s.test_id = $1 AND t.user_id = $2 AND s.skill = $3;
