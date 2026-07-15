package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UpdateCategory struct {
	UserID uuid.UUID
	ID     uuid.UUID
	Name   string
	Color  string
}

type UpdateHandler struct {
	q *db.Queries
}

func NewUpdateHandler(q *db.Queries) UpdateHandler {
	return UpdateHandler{q: q}
}

func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateCategory) (db.Category, error) {
	name := strings.TrimSpace(cmd.Name)

	if name == "" {
		return db.Category{}, ErrEmptyName
	}

	taken, err := h.q.CategoryNameTakenByOther(ctx, db.CategoryNameTakenByOtherParams{
		UserID:    ids.ToPg(cmd.UserID),
		Name:      name,
		ExcludeID: ids.ToPg(cmd.ID),
	})
	if err != nil {
		return db.Category{}, err
	}

	if taken {
		return db.Category{}, ErrNameTaken
	}

	cat, err := h.q.UpdateCategory(ctx, db.UpdateCategoryParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
		Name:   db.Text(&name),
		Color:  cmd.Color,
	})

	if db.IsNotFound(err) {
		return db.Category{}, ErrNotFound
	}

	if err != nil {
		return db.Category{}, err
	}

	return cat, nil
}
