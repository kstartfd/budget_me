package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// SaveBudget sets the limit from ValidFrom onward — the month being edited, and
// every month after it until the user changes it again.
type SaveBudget struct {
	UserID     uuid.UUID
	CategoryID uuid.UUID
	ValidFrom  string
	LimitMinor int64
}

type SaveHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewSaveHandler(q *db.Queries, clk clock.Clock) SaveHandler {
	return SaveHandler{q: q, clock: clk}
}

func (h SaveHandler) Handle(ctx context.Context, cmd SaveBudget) (db.Budget, error) {
	if cmd.LimitMinor <= 0 {
		return db.Budget{}, ErrLimitNotPositive
	}

	visible, err := h.q.CategoryVisibleToUser(ctx, db.CategoryVisibleToUserParams{
		CategoryID: ids.ToPg(cmd.CategoryID),
		UserID:     ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return db.Budget{}, err
	}

	if !visible {
		return db.Budget{}, ErrInvalidCategory
	}

	// Limits apply to expenses only — an income category has no "limit".
	dir, err := h.q.CategoryDirection(ctx, ids.ToPg(cmd.CategoryID))
	if err != nil {
		return db.Budget{}, err
	}

	if dir != db.TxDirectionExpense {
		return db.Budget{}, ErrNotExpense
	}

	return h.q.UpsertBudget(ctx, db.UpsertBudgetParams{
		ID:         ids.ToPg(ids.New()),
		UserID:     ids.ToPg(cmd.UserID),
		CategoryID: ids.ToPg(cmd.CategoryID),
		ValidFrom:  cmd.ValidFrom,
		LimitMinor: db.Int8(&cmd.LimitMinor),
		Now:        db.Time(h.clock.Now()),
	})
}
