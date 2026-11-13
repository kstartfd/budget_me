package commands

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// MaxHoursPerWeek is the most a learner can set as their weekly goal.
const MaxHoursPerWeek = 40

// WeeklyGoalMinutes is the time a program's learner means to study each week,
// in minutes, read from the preferences stored with the program; 0 when they
// gave none. A program made when the survey still asked for minutes a day and
// days a week keeps that answer as its goal, the two multiplied.
func WeeklyGoalMinutes(preferences []byte) int {
	var stored struct {
		HoursPerWeek  int `json:"hoursPerWeek"`
		MinutesPerDay int `json:"minutesPerDay"`
		DaysPerWeek   int `json:"daysPerWeek"`
	}

	if json.Unmarshal(preferences, &stored) != nil {
		return 0
	}

	if stored.HoursPerWeek > 0 && stored.HoursPerWeek <= MaxHoursPerWeek {
		return stored.HoursPerWeek * 60
	}

	if stored.MinutesPerDay > 0 && stored.DaysPerWeek > 0 {
		return stored.MinutesPerDay * stored.DaysPerWeek
	}

	return 0
}

// SetWeeklyGoal sets the hours a week the learner means to study.
type SetWeeklyGoal struct {
	UserID       uuid.UUID
	Language     string
	HoursPerWeek int
}

type GoalHandler struct {
	q *db.Queries
}

func NewGoalHandler(q *db.Queries) GoalHandler {
	return GoalHandler{q: q}
}

// Handle keeps the goal with the running program of the language. Zero takes
// the goal away.
func (h GoalHandler) Handle(ctx context.Context, cmd SetWeeklyGoal) error {
	if cmd.HoursPerWeek < 0 || cmd.HoursPerWeek > MaxHoursPerWeek {
		return apperr.NewInvalid("hours per week must be from 0 to 40")
	}

	n, err := h.q.SetLearningPathWeeklyHours(ctx, db.SetLearningPathWeeklyHoursParams{
		UserID:   ids.ToPg(cmd.UserID),
		Language: cmd.Language,
		Hours:    int32(cmd.HoursPerWeek),
	})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewNotFound("no active program for this language")
	}

	return nil
}
