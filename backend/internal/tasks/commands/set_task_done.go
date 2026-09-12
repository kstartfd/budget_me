package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// SetTaskDone ticks or unticks a task. DoneOn nil unticks it; a set value is
// the LOCAL day the client considers it done — usually the day being viewed,
// not necessarily "today" (ticking while browsing yesterday marks it done on
// yesterday, keeping that day's tally honest).
type SetTaskDone struct {
	UserID uuid.UUID
	ID     uuid.UUID
	DoneOn *string
}

type SetDoneHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewSetDoneHandler(q *db.Queries, clk clock.Clock) SetDoneHandler {
	return SetDoneHandler{q: q, clock: clk}
}

func (h SetDoneHandler) Handle(ctx context.Context, cmd SetTaskDone) (db.Task, error) {
	doneOn, err := db.ParseDateOpt(cmd.DoneOn)
	if err != nil {
		return db.Task{}, ErrInvalidDate
	}

	task, err := h.q.SetTaskDone(ctx, db.SetTaskDoneParams{
		DoneOn:    doneOn,
		UpdatedAt: db.Time(h.clock.Now()),
		ID:        ids.ToPg(cmd.ID),
		UserID:    ids.ToPg(cmd.UserID),
	})
	if db.IsNotFound(err) {
		return db.Task{}, ErrNotFound
	}

	if err != nil {
		return db.Task{}, err
	}

	return task, nil
}
