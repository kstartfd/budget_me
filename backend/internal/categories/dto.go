package categories

import (
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type categoryResponse struct {
	ID        string  `json:"id"`
	Slug      *string `json:"slug,omitempty"`
	Name      *string `json:"name,omitempty"`
	Color     string  `json:"color"`
	Direction string  `json:"direction"` // "income" | "expense"
}

type createCategoryRequest struct {
	Name      string `json:"name"`
	Color     string `json:"color"`
	Direction string `json:"direction"` // "income" | "expense"; empty → expense
}

type updateCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func toCategoryResponse(cat db.Category) categoryResponse {
	return categoryResponse{
		ID:        ids.FromPg(cat.ID).String(),
		Slug:      db.TextPtr(cat.Slug),
		Name:      db.TextPtr(cat.Name),
		Color:     cat.Color,
		Direction: string(cat.Direction),
	}
}
