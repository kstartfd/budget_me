package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UpdateList struct {
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

// Handle can only ever touch a personal list — same WHERE user_id = cmd.UserID
// guard as DeleteHandler, so a preset (user_id IS NULL) can't be renamed this
// way either.
func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateList) (db.ItemList, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return db.ItemList{}, ErrEmptyName
	}

	list, err := h.q.UpdateList(ctx, db.UpdateListParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
		Name:   db.Text(&name),
		Color:  cmd.Color,
	})
	if db.IsNotFound(err) {
		return db.ItemList{}, ErrNotFound
	}

	if err != nil {
		return db.ItemList{}, err
	}

	return list, nil
}
