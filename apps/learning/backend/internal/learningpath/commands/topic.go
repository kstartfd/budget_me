package commands

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// CompleteTopic marks a topic the learner has worked through. Only a topic
// that was started can be completed; the program itself decides nothing here.
type CompleteTopic struct {
	UserID  uuid.UUID
	TopicID uuid.UUID
}

type TopicHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTopicHandler(q *db.Queries, clk clock.Clock) TopicHandler {
	return TopicHandler{q: q, clock: clk}
}

func (h TopicHandler) Complete(ctx context.Context, cmd CompleteTopic) (db.LearningPathTopic, error) {
	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(cmd.TopicID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return db.LearningPathTopic{}, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return db.LearningPathTopic{}, err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(cmd.UserID)})
	if err != nil {
		return db.LearningPathTopic{}, err
	}

	if path.Status != db.LearningPathStatusActive {
		return db.LearningPathTopic{}, apperr.NewConflict("this program is not active")
	}

	if topic.Status == db.LearningPathTopicStatusCompleted {
		return topic, nil
	}

	if topic.Status != db.LearningPathTopicStatusInProgress {
		return db.LearningPathTopic{}, apperr.NewConflict("start the topic before completing it")
	}

	if _, err := h.q.UpdateLearningPathTopicStatus(ctx, db.UpdateLearningPathTopicStatusParams{
		ID:          topic.ID,
		PathID:      topic.PathID,
		Status:      db.LearningPathTopicStatusCompleted,
		CompletedAt: db.Time(h.clock.Now()),
	}); err != nil {
		return db.LearningPathTopic{}, err
	}

	topic.Status = db.LearningPathTopicStatusCompleted
	topic.CompletedAt = db.Time(h.clock.Now())

	return topic, nil
}

// ResetPath archives the active program of a language, keeping its topics,
// exercises and attempts in the history. Starting the next placement test is
// the caller's next call; the two stay separate so a reset cannot half-create
// a test when nothing is running.
type ResetPath struct {
	UserID   uuid.UUID
	Language string
}

type ResetHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewResetHandler(q *db.Queries, clk clock.Clock) ResetHandler {
	return ResetHandler{q: q, clock: clk}
}

func (h ResetHandler) Handle(ctx context.Context, cmd ResetPath) error {
	path, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: ids.ToPg(cmd.UserID), Language: cmd.Language})
	if db.IsNotFound(err) {
		return apperr.NewNotFound("no active program for this language")
	}

	if err != nil {
		return err
	}

	n, err := h.q.ArchiveLearningPath(ctx, db.ArchiveLearningPathParams{
		ID:         path.ID,
		UserID:     path.UserID,
		Status:     db.LearningPathStatusAbandoned,
		ArchivedAt: db.Time(h.clock.Now()),
	})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewConflict("this program is already closed")
	}

	return nil
}

// ClearHistory deletes the closed programs of one language, with their topics,
// exercises, attempts and lessons. The running program stays. What the history
// holds is also what lets a learner start a level test above A1 (the newest
// finished program says which level they confirmed), so after this the next test
// of the language starts from A1.
type ClearHistory struct {
	UserID   uuid.UUID
	Language string
}

type ClearHistoryHandler struct {
	q *db.Queries
}

func NewClearHistoryHandler(q *db.Queries) ClearHistoryHandler {
	return ClearHistoryHandler{q: q}
}

// Handle returns how many programs went.
func (h ClearHistoryHandler) Handle(ctx context.Context, cmd ClearHistory) (int64, error) {
	if cmd.Language == "" {
		return 0, apperr.NewInvalid("language is required")
	}

	return h.q.DeleteLearningPathHistory(ctx, db.DeleteLearningPathHistoryParams{UserID: ids.ToPg(cmd.UserID), Language: cmd.Language})
}
