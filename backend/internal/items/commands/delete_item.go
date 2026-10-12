package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/photostore"
)

type DeleteItem struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	q     *db.Queries
	store photostore.Store
}

func NewDeleteHandler(q *db.Queries, store photostore.Store) DeleteHandler {
	return DeleteHandler{q: q, store: store}
}

// Handle is idempotent — deleting an absent item is a no-op, matching
// categories/lists. DeleteItem returns the deleted row so a photo file
// left on disk gets cleaned up along with it.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteItem) error {
	item, err := h.q.DeleteItem(ctx, db.DeleteItemParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if db.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	if photo := db.TextPtr(item.PhotoPath); photo != nil {
		_ = h.store.Remove(*photo)
	}

	return nil
}
