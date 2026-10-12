package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreateCategory struct {
	UserID    uuid.UUID
	Name      string
	Color     string
	Direction db.TxDirection
}

type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

func (h CreateHandler) Handle(ctx context.Context, cmd CreateCategory) (db.Category, error) {
	name := strings.TrimSpace(cmd.Name)

	if name == "" {
		return db.Category{}, ErrEmptyName
	}

	taken, err := h.q.CategoryNameExists(ctx, db.CategoryNameExistsParams{
		UserID: ids.ToPg(cmd.UserID),
		Name:   name,
	})
	if err != nil {
		return db.Category{}, err
	}

	if taken {
		return db.Category{}, ErrNameTaken
	}

	return h.q.InsertCategory(ctx, db.InsertCategoryParams{
		ID:        ids.ToPg(ids.New()),
		UserID:    ids.ToPg(cmd.UserID),
		Slug:      db.Text(nil),
		Name:      db.Text(&name),
		Color:     cmd.Color,
		CreatedAt: db.Time(h.clock.Now()),
		Direction: cmd.Direction,
	})
}
