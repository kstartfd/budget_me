package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreateTask struct {
	UserID     uuid.UUID
	Title      string
	Note       *string
	PlannedFor string
}

type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

func (h CreateHandler) Handle(ctx context.Context, cmd CreateTask) (db.Task, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Task{}, ErrEmptyTitle
	}

	plannedFor, err := db.ParseDate(cmd.PlannedFor)
	if err != nil {
		return db.Task{}, ErrInvalidDate
	}

	now := h.clock.Now()

	return h.q.InsertTask(ctx, db.InsertTaskParams{
		ID:         ids.ToPg(ids.New()),
		UserID:     ids.ToPg(cmd.UserID),
		Title:      title,
		Note:       db.Text(trimmedOrNil(cmd.Note)),
		PlannedFor: plannedFor,
		CreatedAt:  db.Time(now),
	})
}

// trimmedOrNil trims an optional note and turns an empty result into "no
// note" — so `note: "  "` and `note: null` end up the same row.
func trimmedOrNil(note *string) *string {
	if note == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*note)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
