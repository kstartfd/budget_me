package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ConfirmPlan struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type ConfirmHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewConfirmHandler(q *db.Queries, clk clock.Clock) ConfirmHandler {
	return ConfirmHandler{q: q, clock: clk}
}

// Handle stamps confirmed_at. The transaction itself is created app-side via the
// normal add flow; this just records that the plan happened (kept for history).
func (h ConfirmHandler) Handle(ctx context.Context, cmd ConfirmPlan) (db.Plan, error) {
	plan, err := h.q.ConfirmPlan(ctx, db.ConfirmPlanParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
		Now:    db.Time(h.clock.Now()),
	})
	if db.IsNotFound(err) {
		return db.Plan{}, ErrNotFound
	}

	if err != nil {
		return db.Plan{}, err
	}

	return plan, nil
}
