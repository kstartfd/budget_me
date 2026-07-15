package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreatePlan struct {
	UserID      uuid.UUID
	CategoryID  uuid.UUID
	Month       string
	Title       string
	AmountMinor int64
}

type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

func (h CreateHandler) Handle(ctx context.Context, cmd CreatePlan) (db.Plan, error) {
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

	return h.q.InsertPlan(ctx, db.InsertPlanParams{
		ID:          ids.ToPg(ids.New()),
		UserID:      ids.ToPg(cmd.UserID),
		CategoryID:  ids.ToPg(cmd.CategoryID),
		Month:       cmd.Month,
		Title:       title,
		AmountMinor: cmd.AmountMinor,
		Now:         db.Time(h.clock.Now()),
	})
}
