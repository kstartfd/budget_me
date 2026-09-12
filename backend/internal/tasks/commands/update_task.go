package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// UpdateTask replaces the editable fields wholesale — this is also what an
// explicit "отложить на завтра" is (a plannedFor-only change from the
// caller's point of view), distinct from the carry-over the day query does on
// its own, which never touches a row.
type UpdateTask struct {
	UserID     uuid.UUID
	ID         uuid.UUID
	Title      string
	Note       *string
	PlannedFor string
}

type UpdateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewUpdateHandler(q *db.Queries, clk clock.Clock) UpdateHandler {
	return UpdateHandler{q: q, clock: clk}
}

func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateTask) (db.Task, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Task{}, ErrEmptyTitle
	}

	plannedFor, err := db.ParseDate(cmd.PlannedFor)
	if err != nil {
		return db.Task{}, ErrInvalidDate
	}

	task, err := h.q.UpdateTask(ctx, db.UpdateTaskParams{
		ID:         ids.ToPg(cmd.ID),
		UserID:     ids.ToPg(cmd.UserID),
		Title:      title,
		Note:       db.Text(trimmedOrNil(cmd.Note)),
		PlannedFor: plannedFor,
		UpdatedAt:  db.Time(h.clock.Now()),
	})
	if db.IsNotFound(err) {
		return db.Task{}, ErrNotFound
	}

	if err != nil {
		return db.Task{}, err
	}

	return task, nil
}
