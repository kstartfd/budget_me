package lists

import (
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type listResponse struct {
	ID    string  `json:"id"`
	Slug  *string `json:"slug"`
	Name  *string `json:"name"`
	Color string  `json:"color"`
}

type createListRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func toListResponse(l db.ItemList) listResponse {
	return listResponse{
		ID:    ids.FromPg(l.ID).String(),
		Slug:  db.TextPtr(l.Slug),
		Name:  db.TextPtr(l.Name),
		Color: l.Color,
	}
}
