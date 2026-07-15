
-- name: GetAdvice :one
SELECT * FROM ai_advice WHERE user_id = sqlc.arg(user_id) AND month = sqlc.arg(month);

-- name: UpsertAdvice :exec
INSERT INTO ai_advice (user_id, month, facts_hash, text, created_at)
VALUES (sqlc.arg(user_id), sqlc.arg(month), sqlc.arg(facts_hash), sqlc.arg(text), sqlc.arg(now))
ON CONFLICT (user_id, month)
DO UPDATE SET facts_hash = EXCLUDED.facts_hash,
              text       = EXCLUDED.text,
              created_at = EXCLUDED.created_at;
