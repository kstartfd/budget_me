package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ListBudgets struct {
	UserID uuid.UUID
	Month  string
}

// Budget is the limit in force for the month that was asked about. ValidFrom is
// the month it started applying from, which is usually NOT the month asked about
// — a limit rides forward until something replaces it, so July's answer can be a
// row written in March.
type Budget struct {
	CategoryID uuid.UUID
	ValidFrom  string
	LimitMinor int64
}

type ListHandler struct {
	q *db.Queries
}

func NewListHandler(q *db.Queries) ListHandler {
	return ListHandler{q: q}
}

// Handle drops tombstones. A NULL limit is the row that ENDED a run, so the
// category has no limit this month and belongs in the response no more than one
// that never had a limit at all.
func (h ListHandler) Handle(ctx context.Context, query ListBudgets) ([]Budget, error) {
	rows, err := h.q.ListBudgetsEffectiveIn(ctx, db.ListBudgetsEffectiveInParams{
		UserID: ids.ToPg(query.UserID),
		Month:  query.Month,
	})
	if err != nil {
		return nil, err
	}

	out := make([]Budget, 0, len(rows))

	for _, r := range rows {
		if !r.LimitMinor.Valid {
			continue
		}

		out = append(out, Budget{
			CategoryID: ids.FromPg(r.CategoryID),
			ValidFrom:  r.ValidFrom,
			LimitMinor: r.LimitMinor.Int64,
		})
	}

	return out, nil
}
