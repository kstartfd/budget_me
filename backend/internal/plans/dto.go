package plans

import (
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type planResponse struct {
	ID          string `json:"id"`
	Month       string `json:"month"`
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	AmountMinor int64  `json:"amountMinor"`
	Confirmed   bool   `json:"confirmed"`
}

type planSuggestionResponse struct {
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	AmountMinor int64  `json:"amountMinor"`
	SeenCount   int    `json:"seenCount"`
}

type createPlanRequest struct {
	Month       string `json:"month"`
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	AmountMinor int64  `json:"amountMinor"`
}

type updatePlanRequest struct {
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	AmountMinor int64  `json:"amountMinor"`
}

func toPlanResponse(p db.Plan) planResponse {
	return planResponse{
		ID:          ids.FromPg(p.ID).String(),
		Month:       p.Month,
		CategoryID:  ids.FromPg(p.CategoryID).String(),
		Title:       p.Title,
		AmountMinor: p.AmountMinor,
		Confirmed:   p.ConfirmedAt.Valid,
	}
}

func toPlanSuggestionResponse(s db.ListPlanSuggestionsForMonthRow) planSuggestionResponse {
	return planSuggestionResponse{
		CategoryID:  ids.FromPg(s.CategoryID).String(),
		Title:       s.Title,
		AmountMinor: s.AmountMinor,
		SeenCount:   int(s.SeenCount),
	}
}
