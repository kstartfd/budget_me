package learningpath_test

import (
	"testing"
)

// TestEveryCallOfTheAgentIsCountedByPurpose: the admin reads where the tokens
// go from these sums, so a call must land under what it was for, with the
// amounts the provider reported.
func TestEveryCallOfTheAgentIsCountedByPurpose(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A1"))

	rows, err := e.pool.Query(t.Context(), `SELECT purpose, provider, model, calls, input_tokens, cached_tokens, output_tokens
		FROM learning_agent_usage WHERE user_id = $1`, e.user)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	byPurpose := map[string]struct {
		provider, model              string
		calls, input, cached, output int64
	}{}

	for rows.Next() {
		var (
			purpose string
			row     struct {
				provider, model              string
				calls, input, cached, output int64
			}
		)

		if err := rows.Scan(&purpose, &row.provider, &row.model, &row.calls, &row.input, &row.cached, &row.output); err != nil {
			t.Fatal(err)
		}

		byPurpose[purpose] = row
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	for _, purpose := range []string{"test-questions", "test-judge", "plan"} {
		row, ok := byPurpose[purpose]
		if !ok {
			t.Fatalf("no usage under %q: %+v", purpose, byPurpose)
		}

		// The stub answers as Anthropic with one token in and one out.
		if row.provider != "anthropic" || row.model != "claude-opus-5-5" || row.calls < 1 || row.input != row.calls || row.output != row.calls || row.cached != 0 {
			t.Errorf("%s: %+v", purpose, row)
		}
	}
}
