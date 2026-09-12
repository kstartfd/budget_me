package items

import (
	"time"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type itemResponse struct {
	ID        string  `json:"id"`
	ListID    string  `json:"listId"`
	Title     string  `json:"title"`
	Purpose   *string `json:"purpose"`
	Verdict   string  `json:"verdict"`
	Rating    *int    `json:"rating"`
	Note      *string `json:"note"`
	CreatedAt string  `json:"createdAt"`
}

type createItemRequest struct {
	ListID  string  `json:"listId"`
	Title   string  `json:"title"`
	Purpose *string `json:"purpose"`
	Verdict string  `json:"verdict"`
	Rating  *int    `json:"rating"`
	Note    *string `json:"note"`
}

type updateItemRequest struct {
	Title   string  `json:"title"`
	Purpose *string `json:"purpose"`
	Verdict string  `json:"verdict"`
	Rating  *int    `json:"rating"`
	Note    *string `json:"note"`
}

func toItemResponse(i db.Item) itemResponse {
	return itemResponse{
		ID:        ids.FromPg(i.ID).String(),
		ListID:    ids.FromPg(i.ListID).String(),
		Title:     i.Title,
		Purpose:   db.TextPtr(i.Purpose),
		Verdict:   string(i.Verdict),
		Rating:    db.Int2Ptr(i.Rating),
		Note:      db.TextPtr(i.Note),
		CreatedAt: i.CreatedAt.Time.UTC().Format(time.RFC3339),
	}
}
