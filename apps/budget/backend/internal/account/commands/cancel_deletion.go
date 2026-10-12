package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CancelDeletion struct {
	UserID uuid.UUID
}

type CancelDeletionHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCancelDeletionHandler(q *db.Queries, clk clock.Clock) CancelDeletionHandler {
	return CancelDeletionHandler{q: q, clock: clk}
}

// Handle calls the deletion off. Idempotent: cancelling when nothing is pending
// simply clears a NULL.
func (h CancelDeletionHandler) Handle(ctx context.Context, cmd CancelDeletion) (db.User, error) {
	return h.q.CancelUserDeletion(ctx, db.CancelUserDeletionParams{
		ID:  ids.ToPg(cmd.UserID),
		Now: db.Time(h.clock.Now()),
	})
}
