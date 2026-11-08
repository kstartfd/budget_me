package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/learningpath/placement"
	"learning_app/backend/internal/ratelimit"
)

const maxAttemptRunes = 2000

// ErrThrottled marks an answer check the rate limiter refused: the learner is
// asking the agent faster than allowed, which is worth a 429, not a failure.
var ErrThrottled = errors.New("learning path: the agent is being asked too fast")

// SubmitAttempt checks one exercise answer and stores the attempt. Choice and
// fill are graded on the spot against the saved key; a free translation is
// graded by the learner's agent. Either way, when the answer is wrong the
// agent explains the mistake in the learner's own language, and that
// explanation is what the attempt keeps.
type SubmitAttempt struct {
	UserID     uuid.UUID
	ExerciseID uuid.UUID
	Answer     string
}

type AttemptHandler struct {
	q     *db.Queries
	clock clock.Clock
	agent Agent
	limit *ratelimit.Limiter
}

func NewAttemptHandler(q *db.Queries, clk clock.Clock, agent Agent, limit *ratelimit.Limiter) AttemptHandler {
	return AttemptHandler{q: q, clock: clk, agent: agent, limit: limit}
}

type AttemptResult struct {
	Attempt  db.LearningPathAttempt
	Feedback string
}

func (h AttemptHandler) Handle(ctx context.Context, cmd SubmitAttempt) (AttemptResult, error) {
	answer := strings.TrimSpace(cmd.Answer)
	if answer == "" || utf8.RuneCountInString(answer) > maxAttemptRunes {
		return AttemptResult{}, apperr.NewInvalid("the answer must be 1 to 2000 characters")
	}

	exercise, topic, path, err := h.exercise(ctx, cmd.UserID, cmd.ExerciseID)
	if err != nil {
		return AttemptResult{}, err
	}

	if path.Status != db.LearningPathStatusActive {
		return AttemptResult{}, apperr.NewConflict("this program is not active")
	}

	if topic.Status == db.LearningPathTopicStatusCompleted {
		return AttemptResult{}, apperr.NewConflict("this topic is already completed")
	}

	var options []string
	if len(exercise.Options) > 0 {
		if err := json.Unmarshal(exercise.Options, &options); err != nil {
			return AttemptResult{}, err
		}
	}

	correct, feedback, err := h.review(ctx, cmd.UserID, path, exercise, options, answer)
	if err != nil {
		return AttemptResult{}, err
	}

	attempt, err := h.q.InsertLearningPathAttempt(ctx, db.InsertLearningPathAttemptParams{
		ID:         ids.ToPg(ids.New()),
		ExerciseID: exercise.ID,
		UserID:     ids.ToPg(cmd.UserID),
		Answer:     answer,
		Correct:    correct,
		Feedback:   feedback,
		CreatedAt:  db.Time(h.clock.Now()),
	})
	if err != nil {
		return AttemptResult{}, err
	}

	return AttemptResult{Attempt: attempt, Feedback: feedback}, nil
}

// review grades the answer and, when it is wrong or the answer was free,
// asks the agent for the explanation.
func (h AttemptHandler) review(ctx context.Context, userID uuid.UUID, path db.LearningPath, exercise db.LearningPathExercise, options []string, answer string) (bool, string, error) {
	key := placement.Question{Answer: exercise.Answer}

	// Choice and fill have one right answer; only the explanation ever needs
	// the agent.
	if exercise.Kind != db.LearningPathExerciseKindTranslate {
		if placement.Grade(key, answer) {
			return true, "", nil
		}

		feedback, err := h.explain(ctx, userID, path, exercise, options, answer)

		return false, feedback, err
	}

	var verdict review

	learned := learningcommands.LanguageName(path.Language)
	native := learningcommands.LanguageName(path.TranslationLanguage)
	system, prompt := reviewPrompt(learned, native, string(exercise.Kind), exercise.Text, exercise.Prompt, exercise.Answer, answer, options)

	if err := h.gated(ctx, userID, func(callCtx context.Context) error {
		return h.agent.completeJSON(callCtx, userID, system, prompt, &verdict, func() error { return nil })
	}); err != nil {
		return false, "", err
	}

	return verdict.Correct, verdict.Feedback, nil
}

// explain has the agent word the mistake in the learner's own language. The
// learner is told it is wrong either way, so the explanation is all the call
// is for; the agent's own correct flag is ignored.
func (h AttemptHandler) explain(ctx context.Context, userID uuid.UUID, path db.LearningPath, exercise db.LearningPathExercise, options []string, answer string) (string, error) {
	var verdict review

	learned := learningcommands.LanguageName(path.Language)
	native := learningcommands.LanguageName(path.TranslationLanguage)
	system, prompt := reviewPrompt(learned, native, string(exercise.Kind), exercise.Text, exercise.Prompt, exercise.Answer, answer, options)

	if err := h.gated(ctx, userID, func(callCtx context.Context) error {
		return h.agent.completeJSON(callCtx, userID, system, prompt, &verdict, func() error { return nil })
	}); err != nil {
		return "", err
	}

	return verdict.Feedback, nil
}

// gated paces the calls that spend the learner's key; objective grading runs
// outside it, free.
func (h AttemptHandler) gated(ctx context.Context, userID uuid.UUID, call func(ctx context.Context) error) error {
	release, _, ok := h.limit.Acquire(userID)
	if !ok {
		return ErrThrottled
	}

	defer release()

	return call(ctx)
}

// exercise loads an exercise with the topic and path it belongs to, checking
// the path is the learner's on the way.
func (h AttemptHandler) exercise(ctx context.Context, userID, exerciseID uuid.UUID) (db.LearningPathExercise, db.LearningPathTopic, db.LearningPath, error) {
	exercise, err := h.q.GetLearningPathExercise(ctx, db.GetLearningPathExerciseParams{ID: ids.ToPg(exerciseID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return db.LearningPathExercise{}, db.LearningPathTopic{}, db.LearningPath{}, apperr.NewNotFound("exercise not found")
	}

	if err != nil {
		return db.LearningPathExercise{}, db.LearningPathTopic{}, db.LearningPath{}, err
	}

	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: exercise.TopicID, UserID: ids.ToPg(userID)})
	if err != nil {
		return db.LearningPathExercise{}, db.LearningPathTopic{}, db.LearningPath{}, err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return db.LearningPathExercise{}, db.LearningPathTopic{}, db.LearningPath{}, err
	}

	return exercise, topic, path, nil
}
