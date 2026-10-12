package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeletePlan struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

// Handle is idempotent — deleting an absent plan is a no-op.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeletePlan) error {
	return h.q.DeletePlan(ctx, db.DeletePlanParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
}
