package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreateList struct {
	UserID uuid.UUID
	Name   string
	Color  string
}

type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

func (h CreateHandler) Handle(ctx context.Context, cmd CreateList) (db.ItemList, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return db.ItemList{}, ErrEmptyName
	}

	return h.q.InsertList(ctx, db.InsertListParams{
		ID:        ids.ToPg(ids.New()),
		UserID:    ids.ToPg(cmd.UserID),
		Name:      db.Text(&name),
		Color:     cmd.Color,
		CreatedAt: db.Time(h.clock.Now()),
	})
}
