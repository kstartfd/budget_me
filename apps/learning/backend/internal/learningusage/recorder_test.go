package learningusage_test

import (
	"testing"
	"time"

	"learning_app/backend/internal/learningusage"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/testdb"
)

type moving struct{ at time.Time }

func (m *moving) Now() time.Time { return m.at }

// TestCallsAreSummedPerHour: the cost of a call can depend on the hour it was
// made in (a provider's peak hours), so two calls of one hour make one row and a
// call of the next hour a second.
func TestCallsAreSummedPerHour(t *testing.T) {
	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)

	clk := &moving{at: time.Date(2026, 11, 13, 7, 20, 0, 0, time.UTC)}
	rec := learningusage.NewRecorder(q, clk)
	call := llm.Call{UserID: user, Purpose: llm.PurposeChat, Provider: llm.DeepSeek, Model: "deepseek-flash", Usage: llm.Usage{Input: 10, Cached: 20, Output: 5}}

	rec.Record(t.Context(), call)

	clk.at = clk.at.Add(30 * time.Minute)
	rec.Record(t.Context(), call)

	clk.at = clk.at.Add(20 * time.Minute)
	rec.Record(t.Context(), call)

	rows, err := pool.Query(t.Context(), `SELECT hour, calls, input_tokens, cached_tokens, output_tokens
		FROM learning_agent_usage WHERE user_id = $1 ORDER BY hour`, user)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type row struct {
		hour                   time.Time
		calls, in, cached, out int64
	}

	var got []row

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.hour, &r.calls, &r.in, &r.cached, &r.out); err != nil {
			t.Fatal(err)
		}

		got = append(got, r)
	}

	if len(got) != 2 {
		t.Fatalf("want two hours, got %+v", got)
	}

	if !got[0].hour.Equal(time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)) || got[0].calls != 2 || got[0].in != 20 || got[0].cached != 40 || got[0].out != 10 {
		t.Errorf("07:00 must hold the two calls: %+v", got[0])
	}

	if !got[1].hour.Equal(time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)) || got[1].calls != 1 {
		t.Errorf("08:00 must hold the third: %+v", got[1])
	}
}
