package commands

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// SetReminder switches the daily review reminder on or off. The time zone is the
// learner's own (their browser's), so the reminder arrives in their morning.
type SetReminder struct {
	UserID   uuid.UUID
	Enabled  bool
	Timezone string
}

type ReminderHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewReminderHandler(q *db.Queries, clk clock.Clock) ReminderHandler {
	return ReminderHandler{q: q, clock: clk}
}

// Get returns the learner's reminder, or nil when they never set one.
func (h ReminderHandler) Get(ctx context.Context, userID uuid.UUID) (*db.LearningReminder, error) {
	row, err := h.q.GetLearningReminder(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &row, nil
}

func (h ReminderHandler) Set(ctx context.Context, cmd SetReminder) (db.LearningReminder, error) {
	zone := strings.TrimSpace(cmd.Timezone)
	if _, err := time.LoadLocation(zone); err != nil || zone == "" || zone == "Local" {
		return db.LearningReminder{}, apperr.NewInvalid("timezone must be an IANA zone such as Europe/Minsk")
	}

	return h.q.UpsertLearningReminder(ctx, db.UpsertLearningReminderParams{
		UserID: ids.ToPg(cmd.UserID), Enabled: cmd.Enabled, Timezone: zone, CreatedAt: db.Time(h.clock.Now()),
	})
}
