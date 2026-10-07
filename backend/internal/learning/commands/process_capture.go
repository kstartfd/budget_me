package commands

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type ProcessHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
}

func NewProcessHandler(pool *pgxpool.Pool, clk clock.Clock) ProcessHandler {
	return ProcessHandler{pool: pool, q: db.New(pool), clock: clk}
}

func (h ProcessHandler) Handle(ctx context.Context) (bool, error) {
	now := h.clock.Now()

	job, err := h.q.ClaimLearningCapture(ctx, db.ClaimLearningCaptureParams{Now: db.Time(now), LockedUntil: db.Time(now.Add(time.Minute)), LeaseID: ids.ToPg(ids.New())})
	if db.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	err = h.process(ctx, job)
	if err == nil {
		return true, nil
	}

	status := db.LearningCaptureStatusPending
	message := "Temporary processing error"

	var appErr *apperr.Error
	if errors.As(err, &appErr) || job.Attempts >= 5 {
		status = db.LearningCaptureStatusFailed

		message = "Unable to save capture"
		if appErr != nil {
			message = appErr.Msg
		}
	}

	retryErr := h.q.RetryLearningCapture(ctx, db.RetryLearningCaptureParams{ID: job.ID, LeaseID: job.LeaseID, Status: status, AvailableAt: db.Time(now.Add(time.Duration(1<<min(job.Attempts, 5)) * time.Second)), Error: message, UpdatedAt: db.Time(h.clock.Now())})
	if retryErr != nil {
		return true, errors.Join(err, retryErr)
	}

	return true, err
}

func (h ProcessHandler) process(ctx context.Context, job db.LearningCapture) error {
	if job.Attempts > 5 {
		return apperr.NewInvalid("Retry limit exceeded")
	}

	var fields EntryFields
	if err := json.Unmarshal(job.Payload, &fields); err != nil {
		return apperr.NewInvalid("Invalid capture payload")
	}

	fields, err := fields.Normalize()
	if err != nil {
		return err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(context.Background()) }()

	q := h.q.WithTx(tx)
	if _, err := q.LockLearningCaptureLease(ctx, db.LockLearningCaptureLeaseParams{ID: job.ID, LeaseID: job.LeaseID}); err != nil {
		return err
	}

	if _, err := q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(fields.TopicID), UserID: job.UserID}); err != nil {
		if db.IsNotFound(err) {
			return apperr.NewNotFound("Topic was deleted")
		}

		return err
	}

	now := db.Time(h.clock.Now())

	entry, err := q.InsertLearningEntry(ctx, db.InsertLearningEntryParams{ID: job.ID, UserID: job.UserID, TopicID: ids.ToPg(fields.TopicID), Kind: fields.Kind, Text: fields.Text, Transcription: fields.Transcription, Translation: fields.Translation, Explanation: fields.Explanation, NativeUsage: fields.NativeUsage, Example: fields.Example, Rule: fields.Rule, Source: fields.Source, Status: fields.Status, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return err
	}

	if err := q.CompleteLearningCapture(ctx, db.CompleteLearningCaptureParams{ID: job.ID, LeaseID: job.LeaseID, EntryID: entry.ID, UpdatedAt: now}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h ProcessHandler) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			workCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			worked, err := h.Handle(workCtx)

			cancel()

			if err != nil && ctx.Err() == nil {
				slog.Error("learning capture processing", "err", err)
			}

			delay := time.Second
			if worked && err == nil {
				delay = 0
			}

			timer.Reset(delay)
		}
	}
}
