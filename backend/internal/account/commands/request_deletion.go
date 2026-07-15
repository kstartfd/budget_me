package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type RequestDeletion struct {
	UserID uuid.UUID
}

type RequestDeletionHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewRequestDeletionHandler(q *db.Queries, clk clock.Clock) RequestDeletionHandler {
	return RequestDeletionHandler{q: q, clock: clk}
}

// Handle marks the account for deletion. Nothing is removed here — cmd/worker
// sweeps the row once the grace period is up. Asking twice keeps the first
// deadline (the SQL COALESCEs), so a double tap cannot extend the window.
func (h RequestDeletionHandler) Handle(ctx context.Context, cmd RequestDeletion) (db.User, error) {
	return h.q.RequestUserDeletion(ctx, db.RequestUserDeletionParams{
		ID:  ids.ToPg(cmd.UserID),
		Now: db.Time(h.clock.Now()),
	})
}
