package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type DayTasks struct {
	UserID uuid.UUID
	Day    string
}

type DayListHandler struct {
	q *db.Queries
}

func NewDayListHandler(q *db.Queries) DayListHandler {
	return DayListHandler{q: q}
}

func (h DayListHandler) Handle(ctx context.Context, query DayTasks) ([]db.Task, error) {
	day, err := db.ParseDate(query.Day)
	if err != nil {
		return nil, apperr.NewInvalid("day must be YYYY-MM-DD")
	}

	return h.q.ListTasksForDay(ctx, db.ListTasksForDayParams{
		UserID: ids.ToPg(query.UserID),
		Day:    day,
	})
}
