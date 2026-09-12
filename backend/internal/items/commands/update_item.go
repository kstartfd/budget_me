package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UpdateItem struct {
	UserID  uuid.UUID
	ID      uuid.UUID
	Title   string
	Purpose *string
	Verdict db.ItemVerdict
	Rating  *int
	Note    *string
}

type UpdateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewUpdateHandler(q *db.Queries, clk clock.Clock) UpdateHandler {
	return UpdateHandler{q: q, clock: clk}
}

func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateItem) (db.Item, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Item{}, ErrEmptyTitle
	}

	if err := checkRating(cmd.Rating); err != nil {
		return db.Item{}, err
	}

	verdict := cmd.Verdict
	if verdict == "" {
		verdict = db.ItemVerdictUnknown
	}

	item, err := h.q.UpdateItem(ctx, db.UpdateItemParams{
		ID:        ids.ToPg(cmd.ID),
		UserID:    ids.ToPg(cmd.UserID),
		Title:     title,
		Purpose:   db.Text(cmd.Purpose),
		Verdict:   verdict,
		Rating:    db.Int2(cmd.Rating),
		Note:      db.Text(cmd.Note),
		UpdatedAt: db.Time(h.clock.Now()),
	})
	if db.IsNotFound(err) {
		return db.Item{}, ErrNotFound
	}

	if err != nil {
		return db.Item{}, err
	}

	return item, nil
}
