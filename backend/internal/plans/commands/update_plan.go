package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UpdatePlan struct {
	UserID      uuid.UUID
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Title       string
	AmountMinor int64
}

type UpdateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewUpdateHandler(q *db.Queries, clk clock.Clock) UpdateHandler {
	return UpdateHandler{q: q, clock: clk}
}

func (h UpdateHandler) Handle(ctx context.Context, cmd UpdatePlan) (db.Plan, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Plan{}, ErrEmptyTitle
	}

	if cmd.AmountMinor <= 0 {
		return db.Plan{}, ErrAmountNotPositive
	}

	if err := checkExpenseCategory(ctx, h.q, cmd.UserID, cmd.CategoryID); err != nil {
		return db.Plan{}, err
	}

	plan, err := h.q.UpdatePlan(ctx, db.UpdatePlanParams{
		ID:          ids.ToPg(cmd.ID),
		UserID:      ids.ToPg(cmd.UserID),
		CategoryID:  ids.ToPg(cmd.CategoryID),
		Title:       title,
		AmountMinor: cmd.AmountMinor,
		Now:         db.Time(h.clock.Now()),
	})
	if db.IsNotFound(err) {
		return db.Plan{}, ErrNotFound
	}

	if err != nil {
		return db.Plan{}, err
	}

	return plan, nil
}
