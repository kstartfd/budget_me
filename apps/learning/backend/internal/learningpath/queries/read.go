package queries

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// PathWithTopics is a program and its steps, what the program screen renders.
type PathWithTopics struct {
	Path   db.LearningPath
	Topics []db.LearningPathTopic
}

// TopicDetail is one step of the program with its exercises and every attempt
// the learner has made on them.
type TopicDetail struct {
	Path      db.LearningPath
	Topic     db.LearningPathTopic
	Exercises []db.LearningPathExercise
	Attempts  []db.LearningPathAttempt
}

type ReadHandler struct {
	q *db.Queries
}

func NewReadHandler(q *db.Queries) ReadHandler {
	return ReadHandler{q: q}
}

// Active returns the running program of a language with its topics, or
// NotFound when there is none.
func (h ReadHandler) Active(ctx context.Context, userID uuid.UUID, language string) (PathWithTopics, error) {
	path, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: ids.ToPg(userID), Language: language})
	if db.IsNotFound(err) {
		return PathWithTopics{}, apperr.NewNotFound("no active program for this language")
	}

	if err != nil {
		return PathWithTopics{}, err
	}

	topics, err := h.q.ListLearningPathTopics(ctx, path.ID)
	if err != nil {
		return PathWithTopics{}, err
	}

	return PathWithTopics{Path: path, Topics: topics}, nil
}

// History returns every closed program, newest first, each with its topics.
func (h ReadHandler) History(ctx context.Context, userID uuid.UUID) ([]PathWithTopics, error) {
	paths, err := h.q.ListLearningPathHistory(ctx, ids.ToPg(userID))
	if err != nil {
		return nil, err
	}

	out := make([]PathWithTopics, len(paths))
	for i, path := range paths {
		topics, err := h.q.ListLearningPathTopics(ctx, path.ID)
		if err != nil {
			return nil, err
		}

		out[i] = PathWithTopics{Path: path, Topics: topics}
	}

	return out, nil
}

// Topic returns one topic with its path, exercises and the learner's attempts.
func (h ReadHandler) Topic(ctx context.Context, userID, topicID uuid.UUID) (TopicDetail, error) {
	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return TopicDetail{}, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return TopicDetail{}, err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return TopicDetail{}, err
	}

	exercises, err := h.q.ListLearningPathExercises(ctx, topic.ID)
	if err != nil {
		return TopicDetail{}, err
	}

	attempts, err := h.q.ListLearningPathAttemptsByTopic(ctx, db.ListLearningPathAttemptsByTopicParams{TopicID: topic.ID, UserID: ids.ToPg(userID)})
	if err != nil {
		return TopicDetail{}, err
	}

	return TopicDetail{Path: path, Topic: topic, Exercises: exercises, Attempts: attempts}, nil
}
