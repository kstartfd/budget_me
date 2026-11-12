package queries

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// ExerciseCount is what a topic's current exercise set holds and how much of it
// the learner has answered right.
type ExerciseCount struct {
	Total int
	Done  int
}

// PathWithTopics is a program and its steps, what the program screen renders.
// Exercises counts the current set of each topic that has one.
type PathWithTopics struct {
	Path      db.LearningPath
	Topics    []db.LearningPathTopic
	Exercises map[uuid.UUID]ExerciseCount
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

	return h.withTopics(ctx, userID, path)
}

// withTopics adds the steps of a program, and what each one's exercises hold.
func (h ReadHandler) withTopics(ctx context.Context, userID uuid.UUID, path db.LearningPath) (PathWithTopics, error) {
	topics, err := h.q.ListLearningPathTopics(ctx, path.ID)
	if err != nil {
		return PathWithTopics{}, err
	}

	counts, err := h.q.CountLearningPathExercises(ctx, db.CountLearningPathExercisesParams{PathID: path.ID, UserID: ids.ToPg(userID)})
	if err != nil {
		return PathWithTopics{}, err
	}

	exercises := make(map[uuid.UUID]ExerciseCount, len(counts))
	for _, c := range counts {
		exercises[ids.FromPg(c.TopicID)] = ExerciseCount{Total: int(c.Total), Done: int(c.Done)}
	}

	return PathWithTopics{Path: path, Topics: topics, Exercises: exercises}, nil
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

	return h.withTopics(ctx, userID, path)
}

// ExerciseSpeech is what a learner may hear of an exercise, and in which
// language: the sentence of a listening exercise, or the phrase to repeat in a
// speaking one. Any other exercise has nothing to listen to.
func (h ReadHandler) ExerciseSpeech(ctx context.Context, userID, exerciseID uuid.UUID) (string, string, error) {
	exercise, err := h.q.GetLearningPathExercise(ctx, db.GetLearningPathExerciseParams{ID: ids.ToPg(exerciseID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return "", "", apperr.NewNotFound("exercise not found")
	}

	if err != nil {
		return "", "", err
	}

	var text string

	switch exercise.Kind {
	case db.LearningPathExerciseKindListen:
		text = exercise.Text
	case db.LearningPathExerciseKindSpeak:
		text = exercise.Answer
	default:
		return "", "", apperr.NewInvalid("this exercise has nothing to listen to")
	}

	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: exercise.TopicID, UserID: ids.ToPg(userID)})
	if err != nil {
		return "", "", err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return "", "", err
	}

	return path.Language, text, nil
}

// WordSpeech is a key word of a topic, to be heard, with its language.
func (h ReadHandler) WordSpeech(ctx context.Context, userID, topicID uuid.UUID, index int) (string, string, error) {
	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return "", "", apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return "", "", err
	}

	var words []struct {
		Word string `json:"word"`
	}

	if err := json.Unmarshal(topic.Vocabulary, &words); err != nil || index < 0 || index >= len(words) {
		return "", "", apperr.NewNotFound("word not found")
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return "", "", err
	}

	return path.Language, words[index].Word, nil
}

// History returns every closed program, newest first, each with its topics.
func (h ReadHandler) History(ctx context.Context, userID uuid.UUID) ([]PathWithTopics, error) {
	paths, err := h.q.ListLearningPathHistory(ctx, ids.ToPg(userID))
	if err != nil {
		return nil, err
	}

	out := make([]PathWithTopics, len(paths))
	for i, path := range paths {
		out[i], err = h.withTopics(ctx, userID, path)
		if err != nil {
			return nil, err
		}
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
