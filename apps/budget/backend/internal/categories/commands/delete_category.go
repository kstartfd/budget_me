package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DeleteCategory struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteCategory) error {
	n, err := h.q.DeleteCategory(ctx, db.DeleteCategoryParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrNotFound
	}

	return nil
}
