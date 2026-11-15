package queries

import (
	"context"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type ReadHandler struct{ q *db.Queries }

func NewReadHandler(q *db.Queries) ReadHandler { return ReadHandler{q: q} }

func (h ReadHandler) Topics(ctx context.Context, userID uuid.UUID) ([]db.LearningTopic, error) {
	return h.q.ListLearningTopics(ctx, ids.ToPg(userID))
}
func (h ReadHandler) Topic(ctx context.Context, userID, topicID uuid.UUID) (db.LearningTopic, error) {
	row, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("topic not found")
	}

	return row, err
}

func (h ReadHandler) Entries(ctx context.Context, userID, topicID uuid.UUID, limit, offset int32, kind, status, search string) ([]db.LearningEntry, error) {
	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)}); err != nil {
		if db.IsNotFound(err) {
			return nil, apperr.NewNotFound("topic not found")
		}

		return nil, err
	}

	return h.q.ListLearningEntries(ctx, db.ListLearningEntriesParams{UserID: ids.ToPg(userID), TopicID: ids.ToPg(topicID), PageLimit: limit, PageOffset: offset, KindFilter: kind, StatusFilter: status, Search: search})
}
// Due is what a topic holds for review at `now`: the counts of what is
// waiting and the next `limit` cards.
type Due struct {
	Overdue int64
	Fresh   int64
	Entries []db.LearningEntry
}

func (h ReadHandler) Due(ctx context.Context, userID, topicID uuid.UUID, now time.Time, limit int32) (Due, error) {
	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)}); err != nil {
		if db.IsNotFound(err) {
			return Due{}, apperr.NewNotFound("topic not found")
		}

		return Due{}, err
	}

	counts, err := h.q.CountDueLearningEntries(ctx, db.CountDueLearningEntriesParams{UserID: ids.ToPg(userID), TopicID: ids.ToPg(topicID), NextReviewAt: db.Time(now)})
	if err != nil {
		return Due{}, err
	}

	rows, err := h.q.ListDueLearningEntries(ctx, db.ListDueLearningEntriesParams{UserID: ids.ToPg(userID), TopicID: ids.ToPg(topicID), NextReviewAt: db.Time(now), Limit: limit})
	if err != nil {
		return Due{}, err
	}

	return Due{Overdue: counts.Overdue, Fresh: counts.Fresh, Entries: rows}, nil
}

func (h ReadHandler) Entry(ctx context.Context, userID, id uuid.UUID) (db.LearningEntry, error) {
	row, err := h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	return row, err
}
func (h ReadHandler) Capture(ctx context.Context, userID, id uuid.UUID) (db.LearningCapture, error) {
	row, err := h.q.GetLearningCapture(ctx, db.GetLearningCaptureParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("capture not found")
	}

	return row, err
}
