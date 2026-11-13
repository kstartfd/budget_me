// Package learningusage keeps count of what each learner's agent uses: tokens
// per day, provider, model and purpose. The llm client tells the recorder of
// every call it makes; the admin reads the sums. Nothing of the prompts or the
// replies is kept, and the amounts are tokens, not money (see migration 0029).
package learningusage

import (
	"log/slog"
	"time"

	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
)

// Recorder is the llm.Recorder that writes to the learning database.
type Recorder struct {
	q     *db.Queries
	clock clock.Clock
}

func NewRecorder(q *db.Queries, clk clock.Clock) Recorder {
	return Recorder{q: q, clock: clk}
}

// Record adds a call to the sums of its day (UTC). A failure is logged and
// swallowed: counting must never fail the learner's call.
func (r Recorder) Record(ctx context.Context, call llm.Call) {
	now := r.clock.Now()
	day := pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Valid: true}

	err := r.q.AddLearningAgentUsage(ctx, db.AddLearningAgentUsageParams{
		UserID:       ids.ToPg(call.UserID),
		Day:          day,
		Provider:     string(call.Provider),
		Model:        call.Model,
		Purpose:      call.Purpose,
		InputTokens:  int64(call.Usage.Input),
		CachedTokens: int64(call.Usage.Cached),
		OutputTokens: int64(call.Usage.Output),
	})
	if err != nil {
		slog.Warn("learning agent usage was not recorded", "user_id", call.UserID, "purpose", call.Purpose, "err", err)
	}
}
