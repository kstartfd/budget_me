package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/photostore"
)

type DeleteList struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

type DeleteHandler struct {
	pool  *pgxpool.Pool
	store photostore.Store
}

func NewDeleteHandler(pool *pgxpool.Pool, store photostore.Store) DeleteHandler {
	return DeleteHandler{pool: pool, store: store}
}

// Handle removes a personal list's children in order, then the list. The
// transaction prevents a partial deletion if any database operation fails.
// Photos are removed only after commit, as with single-item deletion.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteList) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the parent before reading children. The user guard also makes a
	// missing, foreign or shared preset list an idempotent no-op.
	var found int
	err = tx.QueryRow(ctx, `SELECT 1 FROM item_lists WHERE id = $1 AND user_id = $2 FOR UPDATE`, cmd.ID, cmd.UserID).Scan(&found)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	q := db.New(tx)
	items, err := q.ListItemsForList(ctx, db.ListItemsForListParams{
		ListID: ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return err
	}

	var photos []string
	for _, item := range items {
		deleted, err := q.DeleteItem(ctx, db.DeleteItemParams{ID: item.ID, UserID: ids.ToPg(cmd.UserID)})
		if err != nil {
			return err
		}
		if photo := db.TextPtr(deleted.PhotoPath); photo != nil {
			photos = append(photos, *photo)
		}
	}

	_, err = q.DeleteList(ctx, db.DeleteListParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	for _, photo := range photos {
		_ = h.store.Remove(photo)
	}

	return nil
}
