package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ListCategories struct {
	UserID uuid.UUID
}

type ListHandler struct {
	q *db.Queries
}

func NewListHandler(q *db.Queries) ListHandler {
	return ListHandler{q: q}
}

func (h ListHandler) Handle(ctx context.Context, query ListCategories) ([]db.Category, error) {
	return h.q.ListCategoriesForUser(ctx, ids.ToPg(query.UserID))
}
