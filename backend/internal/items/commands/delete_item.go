package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeleteItem struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteItem) error {
	_, err := h.q.DeleteItem(ctx, db.DeleteItemParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})

	return err
}
