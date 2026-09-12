package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeleteTask struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

// Handle is idempotent — deleting an absent task is a no-op, matching
// categories/plans.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteTask) error {
	_, err := h.q.DeleteTask(ctx, db.DeleteTaskParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})

	return err
}
