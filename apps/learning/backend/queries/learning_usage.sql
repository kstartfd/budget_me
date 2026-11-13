-- name: AddLearningAgentUsage :exec
INSERT INTO learning_agent_usage (user_id, hour, provider, model, purpose, calls, input_tokens, cached_tokens, output_tokens)
VALUES ($1, $2, $3, $4, $5, 1, $6, $7, $8)
ON CONFLICT (user_id, hour, provider, model, purpose) DO UPDATE SET
    calls = learning_agent_usage.calls + 1,
    input_tokens = learning_agent_usage.input_tokens + EXCLUDED.input_tokens,
    cached_tokens = learning_agent_usage.cached_tokens + EXCLUDED.cached_tokens,
    output_tokens = learning_agent_usage.output_tokens + EXCLUDED.output_tokens;
