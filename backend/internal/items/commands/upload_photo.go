package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/photostore"
)

type UploadItemPhoto struct {
	UserID uuid.UUID
	ID     uuid.UUID
	Data   []byte
}

type UploadPhotoHandler struct {
	q     *db.Queries
	clock clock.Clock
	store photostore.Store
}

func NewUploadPhotoHandler(q *db.Queries, clk clock.Clock, store photostore.Store) UploadPhotoHandler {
	return UploadPhotoHandler{q: q, clock: clk, store: store}
}

// Handle compresses the upload and swaps it onto the item. The old file (if
// any) is only removed once the DB write has succeeded, so a failed update
// never leaves the item pointing at a photo that no longer exists.
func (h UploadPhotoHandler) Handle(ctx context.Context, cmd UploadItemPhoto) (db.Item, error) {
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

	rel, err := h.store.Save("items", cmd.Data)
	if err != nil {
		return db.Item{}, ErrInvalidPhoto
	}

	item, err := h.q.UpdateItemPhoto(ctx, db.UpdateItemPhotoParams{
		ID:        ids.ToPg(cmd.ID),
		UserID:    ids.ToPg(cmd.UserID),
		PhotoPath: db.Text(&rel),
		UpdatedAt: db.Time(h.clock.Now()),
	})
	if err != nil {
		_ = h.store.Remove(rel)

		return db.Item{}, err
	}

	if old := db.TextPtr(existing.PhotoPath); old != nil {
		_ = h.store.Remove(*old)
	}

	return item, nil
}
