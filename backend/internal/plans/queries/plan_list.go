package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ListPlans struct {
	UserID uuid.UUID
	Month  string
}

type ListHandler struct {
	q *db.Queries
}

func NewListHandler(q *db.Queries) ListHandler {
	return ListHandler{q: q}
}

func (h ListHandler) Handle(ctx context.Context, query ListPlans) ([]db.Plan, error) {
	return h.q.ListPlansForMonth(ctx, db.ListPlansForMonthParams{
		UserID: ids.ToPg(query.UserID),
		Month:  query.Month,
	})
}
