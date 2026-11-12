-- name: DeleteExpiredLearningRulePractice :exec
-- A sitting whose time has run out goes, with the chats about its sentences.
DELETE FROM learning_rule_practice WHERE user_id = $1 AND expires_at <= $2;

-- name: TouchLearningRulePractice :one
-- Opening the page of a rule lifts the expiry that leaving it set.
UPDATE learning_rule_practice
SET expires_at = $3
WHERE user_id = $1 AND rule_id = $2 AND expires_at > $4
RETURNING *;

-- name: LeaveLearningRulePractice :exec
-- Leaving the page of a rule: the sitting stays for a short while, then goes.
UPDATE learning_rule_practice
SET expires_at = LEAST(expires_at, $3)
WHERE user_id = $1 AND rule_id = $2;

-- name: EnsureLearningRulePractice :one
-- The sitting is made with the first thing written into it.
INSERT INTO learning_rule_practice (id, user_id, rule_id, language, translation_language, level, title, theory, created_at, updated_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10)
ON CONFLICT (user_id, rule_id) DO UPDATE SET updated_at = EXCLUDED.updated_at, expires_at = EXCLUDED.expires_at
RETURNING *;

-- name: LockLearningRulePractice :one
SELECT * FROM learning_rule_practice WHERE id = $1 AND user_id = $2 FOR UPDATE;

-- name: SaveLearningRulePracticeExamples :one
UPDATE learning_rule_practice
SET examples = $3, updated_at = $4, expires_at = $5
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SaveLearningRulePracticeExercises :one
UPDATE learning_rule_practice
SET exercises = $3, updated_at = $4, expires_at = $5
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: GetLearningRulePractice :one
SELECT * FROM learning_rule_practice WHERE user_id = $1 AND rule_id = $2;

-- name: GetLearningRulePracticeByID :one
SELECT * FROM learning_rule_practice WHERE id = $1 AND user_id = $2;
