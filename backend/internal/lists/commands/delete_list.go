package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeleteList struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

// Handle is idempotent, matching categories — and, like categories, can only
// ever touch a personal list: the WHERE user_id = cmd.UserID never matches a
// preset row (user_id IS NULL there), so a preset can't be deleted this way.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteList) error {
	_, err := h.q.DeleteList(ctx, db.DeleteListParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})

	return err
}
