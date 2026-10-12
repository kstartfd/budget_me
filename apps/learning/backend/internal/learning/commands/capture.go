package commands

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type EnqueueCapture struct {
	UserID     uuid.UUID
	RequestKey string
	Fields     EntryFields
}
type CaptureHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewCaptureHandler(q *db.Queries, clk clock.Clock) CaptureHandler {
	return CaptureHandler{q: q, clock: clk}
}

func (h CaptureHandler) Handle(ctx context.Context, cmd EnqueueCapture) (db.LearningCapture, error) {
	f, err := cmd.Fields.Normalize()
	if err != nil {
		return db.LearningCapture{}, err
	}

	key := strings.TrimSpace(cmd.RequestKey)
	if len(key) > 128 {
		return db.LearningCapture{}, apperr.NewInvalid("Idempotency-Key exceeds 128 characters")
	}

	if key == "" {
		key = ids.New().String()
	} else {
		// Return the original receipt even if its topic has since been deleted.
		row, err := h.q.GetLearningCaptureByKey(ctx, db.GetLearningCaptureByKeyParams{UserID: ids.ToPg(cmd.UserID), RequestKey: key})
		if err == nil {
			return matchingCapture(row, f)
		}

		if !db.IsNotFound(err) {
			return db.LearningCapture{}, err
		}
	}

	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(f.TopicID), UserID: ids.ToPg(cmd.UserID)}); err != nil {
		if db.IsNotFound(err) {
			return db.LearningCapture{}, apperr.NewNotFound("topic not found")
		}

		return db.LearningCapture{}, err
	}

	payload, err := json.Marshal(f)
	if err != nil {
		return db.LearningCapture{}, err
	}

	row, err := h.q.EnqueueLearningCapture(ctx, db.EnqueueLearningCaptureParams{ID: ids.ToPg(ids.New()), UserID: ids.ToPg(cmd.UserID), RequestKey: key, Payload: payload, AvailableAt: db.Time(h.clock.Now())})
	if err != nil {
		return row, err
	}

	return matchingCapture(row, f)
}

func matchingCapture(row db.LearningCapture, fields EntryFields) (db.LearningCapture, error) {
	var previous EntryFields
	if err := json.Unmarshal(row.Payload, &previous); err != nil {
		return row, err
	}

	if previous != fields {
		return row, apperr.NewConflict("Idempotency-Key already used for another capture")
	}

	return row, nil
}
