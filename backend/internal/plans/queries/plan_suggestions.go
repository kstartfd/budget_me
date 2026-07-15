package queries

import (
	"context"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// historyMonths is how far back a confirmed plan still counts as a habit worth
// suggesting again.
const (
	historyMonths = 3
	monthLayout   = "2006-01"
)

type PlanSuggestions struct {
	UserID uuid.UUID
	Month  string
}

type SuggestionsHandler struct {
	q *db.Queries
}

func NewSuggestionsHandler(q *db.Queries) SuggestionsHandler {
	return SuggestionsHandler{q: q}
}

func (h SuggestionsHandler) Handle(
	ctx context.Context,
	query PlanSuggestions,
) ([]db.ListPlanSuggestionsForMonthRow, error) {
	target, err := time.Parse(monthLayout, query.Month)
	if err != nil {
		return nil, err
	}

	return h.q.ListPlanSuggestionsForMonth(ctx, db.ListPlanSuggestionsForMonthParams{
		UserID:      ids.ToPg(query.UserID),
		TargetMonth: query.Month,
		FromMonth:   target.AddDate(0, -historyMonths, 0).Format(monthLayout),
	})
}
