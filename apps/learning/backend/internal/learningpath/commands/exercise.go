package commands

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	learningcommands "learning_app/backend/internal/learning/commands"
)

// GenerateExercises has the learner's agent write the exercise set of one
// topic and saves it. A topic keeps its current set: reopening returns the
// same exercises, and regenerating adds a version without losing old attempts.
type GenerateExercises struct {
	UserID     uuid.UUID
	TopicID    uuid.UUID
	Regenerate bool
}

type ExerciseHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
	agent Agent
}

func NewExerciseHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, agent Agent) ExerciseHandler {
	return ExerciseHandler{pool: pool, q: q, clock: clk, agent: agent}
}

func (h ExerciseHandler) Handle(ctx context.Context, cmd GenerateExercises) ([]db.LearningPathExercise, error) {
	topic, path, err := h.topic(ctx, cmd.UserID, cmd.TopicID)
	if err != nil {
		return nil, err
	}

	if path.Status != db.LearningPathStatusActive {
		return nil, apperr.NewConflict("this program is not active")
	}

	if topic.Status == db.LearningPathTopicStatusCompleted && !cmd.Regenerate {
		return nil, apperr.NewConflict("this topic is already completed")
	}

	existing, err := h.q.ListLearningPathExercises(ctx, topic.ID)
	if err != nil {
		return nil, err
	}

	if len(existing) > 0 && !cmd.Regenerate {
		return existing, nil
	}
	attempts, err := h.q.ListLearningPathAttemptsByTopic(ctx, db.ListLearningPathAttemptsByTopicParams{
		TopicID: topic.ID,
		UserID:  ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return nil, err
	}
	correct := 0
	for _, attempt := range attempts {
		if attempt.Correct {
			correct++
		}
	}

	learned := learningcommands.LanguageName(path.Language)
	native := learningcommands.LanguageName(path.TranslationLanguage)

	system, prompt := exercisesPrompt(learned, native, path.Level, topic.Skill, topic.Title, topic.Goal, len(attempts), correct)

	var set exerciseSet
	if err := h.agent.completeJSON(ctx, cmd.UserID, system, prompt, &set, func() error {
		return validateExercises(set.Exercises)
	}); err != nil {
		return nil, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)
	now := h.clock.Now()
	if _, err := tq.LockLearningPathTopic(ctx, topic.ID); err != nil {
		return nil, err
	}
	generation, err := tq.GetNextLearningPathExerciseGeneration(ctx, topic.ID)
	if err != nil {
		return nil, err
	}

	out := make([]db.LearningPathExercise, len(set.Exercises))
	for i, e := range set.Exercises {
		raw, err := json.Marshal(e.Options)
		if err != nil {
			return nil, err
		}

		out[i], err = tq.InsertLearningPathExercise(ctx, db.InsertLearningPathExerciseParams{
			ID:        ids.ToPg(ids.New()),
			TopicID:   topic.ID,
			Position:  int32(i),
			Kind:      db.LearningPathExerciseKind(e.Kind),
			Text:      e.Text,
			Prompt:    e.Prompt,
			Options:   raw,
			Answer:    e.Answer,
			CreatedAt: db.Time(now),
		})
		if err != nil {
			return nil, err
		}
		if err := tq.InsertLearningPathExerciseVersion(ctx, db.InsertLearningPathExerciseVersionParams{
			ExerciseID: out[i].ID,
			TopicID:    topic.ID,
			Generation: generation,
		}); err != nil {
			return nil, err
		}
	}

	if topic.Status == db.LearningPathTopicStatusNotStarted || topic.Status == db.LearningPathTopicStatusCompleted {
		// In progress has no completion time; the zero timestamptz is NULL.
		if _, err := tq.UpdateLearningPathTopicStatus(ctx, db.UpdateLearningPathTopicStatusParams{ID: topic.ID, PathID: topic.PathID, Status: db.LearningPathTopicStatusInProgress}); err != nil {
			return nil, err
		}
	}

	return out, tx.Commit(ctx)
}

// topic loads a topic with its path and checks the path is the learner's.
func (h ExerciseHandler) topic(ctx context.Context, userID, topicID uuid.UUID) (db.LearningPathTopic, db.LearningPath, error) {
	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return db.LearningPathTopic{}, db.LearningPath{}, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return db.LearningPathTopic{}, db.LearningPath{}, err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return db.LearningPathTopic{}, db.LearningPath{}, err
	}

	return topic, path, nil
}
