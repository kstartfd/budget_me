package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

// PathByID returns a program of the learner with its topics.
func (h ReadHandler) PathByID(ctx context.Context, userID, pathID uuid.UUID) (PathWithTopics, error) {
	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: ids.ToPg(pathID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return PathWithTopics{}, apperr.NewNotFound("program not found")
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

// TeacherView is a teacher with what its running program looks like: nothing
// when no program runs.
type TeacherView struct {
	Teacher db.LearningTeacher
	Path    *PathSummary
}

// PathSummary is a running program in one line: its level and how far along.
type PathSummary struct {
	ID    uuid.UUID
	Level string
	Done  int
	Total int
}

// Teachers lists the learner's teachers, oldest first, each with the summary of
// its running program.
func (h ReadHandler) Teachers(ctx context.Context, userID uuid.UUID) ([]TeacherView, error) {
	teachers, err := h.q.ListLearningTeachers(ctx, ids.ToPg(userID))
	if err != nil {
		return nil, err
	}

	paths, err := h.q.ListActiveLearningPaths(ctx, ids.ToPg(userID))
	if err != nil {
		return nil, err
	}

	counts := map[uuid.UUID]db.CountLearningPathTopicsRow{}

	if len(paths) > 0 {
		pathIDs := make([]pgtype.UUID, len(paths))
		for i, p := range paths {
			pathIDs[i] = p.ID
		}

		rows, err := h.q.CountLearningPathTopics(ctx, pathIDs)
		if err != nil {
			return nil, err
		}

		for _, r := range rows {
			counts[ids.FromPg(r.PathID)] = r
		}
	}

	byLanguage := map[string]db.LearningPath{}
	for _, p := range paths {
		byLanguage[p.Language] = p
	}

	out := make([]TeacherView, len(teachers))

	for i, t := range teachers {
		out[i] = TeacherView{Teacher: t}

		if p, ok := byLanguage[t.Language]; ok {
			c := counts[ids.FromPg(p.ID)]
			out[i].Path = &PathSummary{ID: ids.FromPg(p.ID), Level: p.Level, Done: int(c.Done), Total: int(c.Total)}
		}
	}

	return out, nil
}
