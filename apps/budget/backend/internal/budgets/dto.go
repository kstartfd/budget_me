package budgets

import (
	"budget_me/backend/internal/budgets/queries"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// ValidFrom is the month the limit started applying from, not the month that was
// asked about — the screen uses it to say "carried over since March" rather than
// implying the user set it again this month.
type budgetResponse struct {
	CategoryID string `json:"categoryId"`
	ValidFrom  string `json:"validFrom"`
	LimitMinor int64  `json:"limitMinor"`
}

type recommendationResponse struct {
	CategoryID string `json:"categoryId"`
	LimitMinor int64  `json:"limitMinor"`
}

// Month is the month being edited; the limit applies from it onward.
type upsertBudgetRequest struct {
	CategoryID string `json:"categoryId"`
	Month      string `json:"month"`
	LimitMinor int64  `json:"limitMinor"`
}

func toBudgetResponse(b queries.Budget) budgetResponse {
	return budgetResponse{
		CategoryID: b.CategoryID.String(),
		ValidFrom:  b.ValidFrom,
		LimitMinor: b.LimitMinor,
	}
}

// The saved row answers with the month it was written for — it IS the start of
// the run, so validFrom and the edited month are the same thing here.
func savedBudgetResponse(b db.Budget) budgetResponse {
	return budgetResponse{
		CategoryID: ids.FromPg(b.CategoryID).String(),
		ValidFrom:  b.ValidFrom,
		LimitMinor: b.LimitMinor.Int64,
	}
}

func toRecommendationResponse(rec queries.Recommendation) recommendationResponse {
	return recommendationResponse{
		CategoryID: rec.CategoryID.String(),
		LimitMinor: rec.LimitMinor,
	}
}
