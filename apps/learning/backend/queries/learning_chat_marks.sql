-- name: GetLearningChatMarks :one
SELECT * FROM learning_chat_marks WHERE user_id = $1;

-- name: UpsertLearningChatMarks :one
INSERT INTO learning_chat_marks (user_id, any_label, dialogue, triggers, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    any_label = EXCLUDED.any_label,
    dialogue = EXCLUDED.dialogue,
    triggers = EXCLUDED.triggers,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteLearningChatMarks :execrows
DELETE FROM learning_chat_marks WHERE user_id = $1;
