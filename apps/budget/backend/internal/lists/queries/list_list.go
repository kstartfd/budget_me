package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ListLists struct {
	UserID uuid.UUID
}

type ListHandler struct {
	q *db.Queries
}

func NewListHandler(q *db.Queries) ListHandler {
	return ListHandler{q: q}
}

func (h ListHandler) Handle(ctx context.Context, query ListLists) ([]db.ItemList, error) {
	return h.q.ListListsForUser(ctx, ids.ToPg(query.UserID))
}
