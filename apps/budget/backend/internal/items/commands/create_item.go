package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreateItem struct {
	UserID  uuid.UUID
	ListID  uuid.UUID
	Title   string
	Purpose *string
	Verdict db.ItemVerdict
	Rating  *int
	Note    *string
}

type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

func (h CreateHandler) Handle(ctx context.Context, cmd CreateItem) (db.Item, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Item{}, ErrEmptyTitle
	}

	if err := checkRating(cmd.Rating); err != nil {
		return db.Item{}, err
	}

	if err := checkListVisible(ctx, h.q, cmd.UserID, cmd.ListID); err != nil {
		return db.Item{}, err
	}

	verdict := cmd.Verdict
	if verdict == "" {
		verdict = db.ItemVerdictUnknown
	}

	now := h.clock.Now()

	return h.q.InsertItem(ctx, db.InsertItemParams{
		ID:        ids.ToPg(ids.New()),
		ListID:    ids.ToPg(cmd.ListID),
		UserID:    ids.ToPg(cmd.UserID),
		Title:     title,
		Purpose:   db.Text(cmd.Purpose),
		Verdict:   verdict,
		Rating:    db.Int2(cmd.Rating),
		Note:      db.Text(cmd.Note),
		CreatedAt: db.Time(now),
	})
}
