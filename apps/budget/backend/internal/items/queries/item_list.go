package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ListItems struct {
	UserID uuid.UUID
	ListID uuid.UUID
}

type ListHandler struct {
	q *db.Queries
}

func NewListHandler(q *db.Queries) ListHandler {
	return ListHandler{q: q}
}

func (h ListHandler) Handle(ctx context.Context, query ListItems) ([]db.Item, error) {
	return h.q.ListItemsForList(ctx, db.ListItemsForListParams{
		ListID: ids.ToPg(query.ListID),
		UserID: ids.ToPg(query.UserID),
	})
}
