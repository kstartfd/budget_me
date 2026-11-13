-- +goose Up
-- What each learner's agent has used, summed per day, provider, model and
-- purpose (what the call was for), so the admin can see where the tokens go.
-- The amounts are tokens, never money: the learner's own API key pays the
-- provider, and a price table at read time turns tokens into an estimate, so a
-- corrected price corrects the history too. Nothing of the prompts is kept.
--   input_tokens:  the prompt read at the full price (what the cache did not serve)
--   cached_tokens: the prompt the provider served from its cache
--   output_tokens: the reply, thinking included
-- Every answered call counts, a retry after an unusable reply as well: it was paid for.
CREATE TABLE learning_agent_usage (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day date NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    purpose text NOT NULL,
    calls integer NOT NULL DEFAULT 0,
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, day, provider, model, purpose)
);
CREATE INDEX learning_agent_usage_day ON learning_agent_usage(day);

-- +goose Down
DROP TABLE learning_agent_usage;
