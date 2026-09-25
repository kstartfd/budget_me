package items

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
	PhotoURL  *string `json:"photoUrl"`
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
		PhotoURL:  photoURL(i.PhotoPath),
		CreatedAt: i.CreatedAt.Time.UTC().Format(time.RFC3339),
	}
}

// photoURL turns a stored relative path ("items/<uuid>.jpg") into the public
// URL the frontend can put straight into an <img src>: public.Static in
// server.go serves internal/photostore's uploads dir at this same prefix.
func photoURL(p pgtype.Text) *string {
	rel := db.TextPtr(p)
	if rel == nil {
		return nil
	}

	url := "/api/v1/uploads/" + *rel

	return &url
}
