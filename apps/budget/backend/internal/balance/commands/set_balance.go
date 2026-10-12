package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type SetBalance struct {
	UserID       uuid.UUID
	BalanceMinor int64
}

type SetHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewSetHandler(q *db.Queries, clk clock.Clock) SetHandler {
	return SetHandler{q: q, clock: clk}
}

// Handle re-anchors the balance to "now": the stated figure is taken as the
// truth at this instant, so transactions before it no longer move it. Negative
// is allowed — an overdrawn card is a real state, not an input error.
func (h SetHandler) Handle(ctx context.Context, cmd SetBalance) (db.User, error) {
	return h.q.SetBalanceAnchor(ctx, db.SetBalanceAnchorParams{
		ID:           ids.ToPg(cmd.UserID),
		BalanceMinor: db.Int8(&cmd.BalanceMinor),
		Now:          db.Time(h.clock.Now()),
	})
}
