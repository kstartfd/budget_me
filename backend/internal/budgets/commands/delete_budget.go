package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// DeleteBudget removes the limit from Month onward. Earlier months keep theirs —
// removing a limit today is not a statement about what it was in March.
type DeleteBudget struct {
	UserID     uuid.UUID
	CategoryID uuid.UUID
	Month      string
}

type DeleteHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewDeleteHandler(q *db.Queries, clk clock.Clock) DeleteHandler {
	return DeleteHandler{q: q, clock: clk}
}

// Handle is idempotent: removing a limit that is already gone is a no-op, which
// suits the optimistic frontend delete (no 404 to surface).
//
// How the removal is recorded depends on what came before. With an earlier run
// behind it, deleting the row would hand the month straight back to that older
// limit — so a tombstone (NULL) is written instead, ending the run while leaving
// the history readable. With nothing earlier there is nothing to inherit and
// nothing to preserve, so the rows just go.
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteBudget) error {
	userID := ids.ToPg(cmd.UserID)
	categoryID := ids.ToPg(cmd.CategoryID)

	inherited, err := h.q.HasBudgetBefore(ctx, db.HasBudgetBeforeParams{
		UserID:     userID,
		CategoryID: categoryID,
		Month:      cmd.Month,
	})
	if err != nil {
		return err
	}

	if !inherited {
		return h.q.DeleteBudgetsFrom(ctx, db.DeleteBudgetsFromParams{
			UserID:     userID,
			CategoryID: categoryID,
			Month:      cmd.Month,
		})
	}

	_, err = h.q.UpsertBudget(ctx, db.UpsertBudgetParams{
		ID:         ids.ToPg(ids.New()),
		UserID:     userID,
		CategoryID: categoryID,
		ValidFrom:  cmd.Month,
		LimitMinor: db.Int8(nil),
		Now:        db.Time(h.clock.Now()),
	})

	return err
}
