package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeleteBudget struct {
	UserID     uuid.UUID
	CategoryID uuid.UUID
	Month      string
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

// Handle is idempotent: removing a limit that is already gone is a no-op, which
// suits the optimistic frontend delete (no 404 to surface).
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteBudget) error {
	return h.q.DeleteBudget(ctx, db.DeleteBudgetParams{
		UserID:     ids.ToPg(cmd.UserID),
		CategoryID: ids.ToPg(cmd.CategoryID),
		Month:      cmd.Month,
	})
}
