package commands

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
)

const (
	// jobTimeout is the whole budget of one build: planning, then the first
	// exercise sets.
	jobTimeout = 12 * time.Minute
	// jobStale is how long a queued or running job may stay silent before it is
	// taken for lost (the server restarted under it) and a new one may start.
	jobStale = 15 * time.Minute
)

// BuildProgram queues the building of a program from a finished placement test.
// It answers at once with a job the screen follows; the work runs in the
// background on the learner's own agent. One job per test: asking again while
// it runs, or after it is done, hands the same job back, and only a failed or
// lost one may be started over.
type BuildProgram struct {
	UserID      uuid.UUID
	TestID      uuid.UUID
	Preferences Preferences
}

type BuildHandler struct {
	q         *db.Queries
	clock     clock.Clock
	analysis  AnalyzeHandler
	exercises ExerciseHandler
	// prefetch is how many of the first topics get their exercises written
	// right after the program is saved, so opening one is instant.
	prefetch int
}

func NewBuildHandler(q *db.Queries, clk clock.Clock, analysis AnalyzeHandler, exercises ExerciseHandler, prefetch int) BuildHandler {
	return BuildHandler{q: q, clock: clk, analysis: analysis, exercises: exercises, prefetch: prefetch}
}

// Enqueue checks what can be checked without the agent and queues the job. The
// bool says whether a new job was made (and so needs running).
func (h BuildHandler) Enqueue(ctx context.Context, cmd BuildProgram) (db.LearningPathJob, bool, error) {
	test, err := h.q.GetLearningPathTest(ctx, db.GetLearningPathTestParams{ID: ids.ToPg(cmd.TestID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return db.LearningPathJob{}, false, apperr.NewNotFound("test not found")
	}

	if err != nil {
		return db.LearningPathJob{}, false, err
	}

	if test.Kind != db.LearningPathTestKindPlacement {
		return db.LearningPathJob{}, false, apperr.NewInvalid("only a placement test builds a program")
	}

	if test.Status != db.LearningPathTestStatusDone {
		return db.LearningPathJob{}, false, apperr.NewConflict("finish the test first")
	}

	// Without an agent there is nothing to queue: say so now, not from a job.
	if _, err := h.q.GetLearningAISettings(ctx, ids.ToPg(cmd.UserID)); db.IsNotFound(err) {
		return db.LearningPathJob{}, false, apperr.NewInvalid("this needs an AI agent: set one up in settings")
	} else if err != nil {
		return db.LearningPathJob{}, false, err
	}

	now := h.clock.Now()

	last, err := h.q.GetLatestLearningPathJobByTest(ctx, db.GetLatestLearningPathJobByTestParams{TestID: ids.ToPg(cmd.TestID), UserID: ids.ToPg(cmd.UserID)})
	if err == nil {
		switch {
		case last.Status == db.LearningPathJobStatusDone:
			return last, false, nil
		case (last.Status == db.LearningPathJobStatusQueued || last.Status == db.LearningPathJobStatusRunning) && now.Sub(last.UpdatedAt.Time) < jobStale:
			return last, false, nil
		}
	} else if !db.IsNotFound(err) {
		return db.LearningPathJob{}, false, err
	}

	if _, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: test.UserID, Language: test.Language}); err == nil {
		return db.LearningPathJob{}, false, apperr.NewConflict("a program for this language is already running: finish it or reset it first")
	} else if !db.IsNotFound(err) {
		return db.LearningPathJob{}, false, err
	}

	prefs, err := json.Marshal(cmd.Preferences.Clean())
	if err != nil {
		return db.LearningPathJob{}, false, err
	}

	job, err := h.q.InsertLearningPathJob(ctx, db.InsertLearningPathJobParams{
		ID:          ids.ToPg(ids.New()),
		UserID:      ids.ToPg(cmd.UserID),
		TestID:      ids.ToPg(cmd.TestID),
		Preferences: prefs,
		CreatedAt:   db.Time(now),
	})

	return job, err == nil, err
}

// Job reads a job of the learner.
func (h BuildHandler) Job(ctx context.Context, userID, jobID uuid.UUID) (db.LearningPathJob, error) {
	job, err := h.q.GetLearningPathJob(ctx, db.GetLearningPathJobParams{ID: ids.ToPg(jobID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return db.LearningPathJob{}, apperr.NewNotFound("job not found")
	}

	return job, err
}

// Run does the job, on its own context: the request that queued it is long
// gone. The program is saved first and the job is done from then on, since the
// learner can start; the first exercise sets are written after, quietly.
func (h BuildHandler) Run(job db.LearningPathJob) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	userID, testID := ids.FromPg(job.UserID), ids.FromPg(job.TestID)

	if err := h.q.SetLearningPathJobRunning(ctx, db.SetLearningPathJobRunningParams{ID: job.ID, UpdatedAt: db.Time(h.clock.Now())}); err != nil {
		slog.Error("learning path job cannot start", "job_id", ids.FromPg(job.ID), "err", err)

		return
	}

	var prefs Preferences
	_ = json.Unmarshal(job.Preferences, &prefs)

	path, topics, err := h.analysis.Handle(ctx, Analyze{UserID: userID, TestID: testID, Preferences: prefs})
	if err != nil {
		message := FailureMessage(err)
		slog.Warn("learning path job failed", "job_id", ids.FromPg(job.ID), "err", err)

		// The context may be the thing that ran out, so the failure is written on a fresh one.
		fail, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()

		_ = h.q.FinishLearningPathJob(fail, db.FinishLearningPathJobParams{ID: job.ID, Status: db.LearningPathJobStatusFailed, Error: message, UpdatedAt: db.Time(h.clock.Now())})

		return
	}

	if err := h.q.FinishLearningPathJob(ctx, db.FinishLearningPathJobParams{ID: job.ID, Status: db.LearningPathJobStatusDone, PathID: path.ID, UpdatedAt: db.Time(h.clock.Now())}); err != nil {
		slog.Error("learning path job cannot finish", "job_id", ids.FromPg(job.ID), "err", err)

		return
	}

	for i := 0; i < min(h.prefetch, len(topics)); i++ {
		if _, err := h.exercises.Handle(ctx, GenerateExercises{UserID: userID, TopicID: ids.FromPg(topics[i].ID), Prefetch: true}); err != nil {
			slog.Warn("learning path exercises were not prepared", "topic_id", ids.FromPg(topics[i].ID), "err", err)

			return
		}
	}
}

// FailureMessage words a failed run the way the learner can act on it.
func FailureMessage(err error) string {
	var app *apperr.Error

	switch {
	case errors.Is(err, llm.ErrRejected):
		return "the provider rejected your API key: enter it again in settings"
	case errors.Is(err, llm.ErrTemporary), errors.Is(err, context.DeadlineExceeded):
		return "the AI agent is not answering, try again"
	case errors.Is(err, ErrUnreadable):
		return "the AI agent's answer could not be read, try again"
	case errors.As(err, &app):
		return app.Msg
	}

	return "the program could not be built, try again"
}
