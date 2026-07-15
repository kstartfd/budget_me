package budgets

import (
	"budget_me/backend/internal/budgets/queries"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type budgetResponse struct {
	CategoryID string `json:"categoryId"`
	Month      string `json:"month"`
	LimitMinor int64  `json:"limitMinor"`
}

type recommendationResponse struct {
	CategoryID string `json:"categoryId"`
	LimitMinor int64  `json:"limitMinor"`
}

type upsertBudgetRequest struct {
	CategoryID string `json:"categoryId"`
	Month      string `json:"month"`
	LimitMinor int64  `json:"limitMinor"`
}

func toBudgetResponse(b db.Budget) budgetResponse {
	return budgetResponse{
		CategoryID: ids.FromPg(b.CategoryID).String(),
		Month:      b.Month,
		LimitMinor: b.LimitMinor,
	}
}

func toRecommendationResponse(rec queries.Recommendation) recommendationResponse {
	return recommendationResponse{
		CategoryID: rec.CategoryID.String(),
		LimitMinor: rec.LimitMinor,
	}
}
