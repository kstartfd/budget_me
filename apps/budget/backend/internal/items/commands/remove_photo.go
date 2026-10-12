package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/photostore"
)

type RemoveItemPhoto struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type RemovePhotoHandler struct {
	q     *db.Queries
	clock clock.Clock
	store photostore.Store
}

func NewRemovePhotoHandler(q *db.Queries, clk clock.Clock, store photostore.Store) RemovePhotoHandler {
	return RemovePhotoHandler{q: q, clock: clk, store: store}
}

func (h RemovePhotoHandler) Handle(ctx context.Context, cmd RemoveItemPhoto) (db.Item, error) {
	existing, err := h.q.GetItemForUser(ctx, db.GetItemForUserParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if db.IsNotFound(err) {
		return db.Item{}, ErrNotFound
	}

	if err != nil {
		return db.Item{}, err
	}

	item, err := h.q.UpdateItemPhoto(ctx, db.UpdateItemPhotoParams{
		ID:        ids.ToPg(cmd.ID),
		UserID:    ids.ToPg(cmd.UserID),
		PhotoPath: db.Text(nil),
		UpdatedAt: db.Time(h.clock.Now()),
	})
	if err != nil {
		return db.Item{}, err
	}

	if old := db.TextPtr(existing.PhotoPath); old != nil {
		_ = h.store.Remove(*old)
	}

	return item, nil
}
