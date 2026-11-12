package commands

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
)

// fillTimeout is the budget of one fill: the learner waits on the button, unlike
// the capture queue, but a card is one short call.
const fillTimeout = 90 * time.Second

// ErrFillFailed marks a call that gave nothing to write from, for a reason that is
// neither a rejected key nor a temporary outage: the provider refused the request,
// or its answer held no field that could be read.
var ErrFillFailed = errors.New("learning: the agent did not give an answer to fill the card from")

// FillEntry has the learner's own agent write the fields of a saved entry that
// are still empty: transcription, translation, explanation, how natives say
// it, an example and the rule behind it. Fields the learner wrote are never
// touched.
type FillEntry struct {
	UserID  uuid.UUID
	EntryID uuid.UUID
}

type FillHandler struct {
	q     *db.Queries
	clock clock.Clock
	vault secretbox.Box
	llm   Completer
}

func NewFillHandler(q *db.Queries, clk clock.Clock, vault secretbox.Box, completer Completer) FillHandler {
	return FillHandler{q: q, clock: clk, vault: vault, llm: completer}
}

// fieldsOf reads the card fields of a stored entry.
func fieldsOf(row db.LearningEntry) EntryFields {
	return EntryFields{
		TopicID: ids.FromPg(row.TopicID), Kind: row.Kind, Text: row.Text,
		Transcription: row.Transcription, Translation: row.Translation, Explanation: row.Explanation,
		NativeUsage: row.NativeUsage, Example: row.Example, Rule: row.Rule,
		Source: row.Source, Status: row.Status,
	}
}

// Handle returns the entry as it is after the fill. An entry with nothing
// missing comes back as it was, without a call.
func (h FillHandler) Handle(ctx context.Context, cmd FillEntry) (db.LearningEntry, error) {
	row, err := h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(cmd.EntryID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	if err != nil {
		return row, err
	}

	want := fieldsOf(row).missing()
	if len(want) == 0 {
		return row, nil
	}

	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(cmd.UserID))
	if db.IsNotFound(err) {
		return row, apperr.NewInvalid("this needs an AI agent: set one up in settings")
	}

	if err != nil {
		return row, err
	}

	topic, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: row.TopicID, UserID: ids.ToPg(cmd.UserID)})
	if err != nil {
		return row, err
	}

	key, err := h.vault.Open(settings.KeyCiphertext, cmd.UserID[:])
	if err != nil {
		slog.Error("learning ai key cannot be decrypted", "user_id", cmd.UserID, "err", err)

		return row, apperr.NewConflict("the saved API key cannot be read; enter it again in settings")
	}

	system, prompt := enrichmentPrompt(fieldsOf(row), topic, want)

	callCtx, cancel := context.WithTimeout(ctx, fillTimeout)
	defer cancel()

	text, err := h.llm.Complete(callCtx, llm.Request{
		Provider: llm.Provider(settings.Provider),
		Model:    settings.Model,
		Level:    string(settings.Level),
		Key:      string(key),
		System:   system,
		Prompt:   prompt,
	})
	if err != nil {
		if errors.Is(err, llm.ErrRejected) || errors.Is(err, llm.ErrTemporary) {
			return row, err
		}

		slog.Warn("learning entry fill: the provider failed", "entry_id", cmd.EntryID, "provider", settings.Provider, "err", err)

		return row, ErrFillFailed
	}

	values := parseEnrichment(text, want)
	if len(values) == 0 {
		slog.Warn("learning entry fill: unreadable reply", "entry_id", cmd.EntryID, "provider", settings.Provider)

		return row, ErrFillFailed
	}

	// The learner may have typed into the card while the agent wrote: fill what is
	// still empty on the entry as it is now, so nothing they wrote is lost.
	row, err = h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(cmd.EntryID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	if err != nil {
		return row, err
	}

	f := fieldsOf(row).fill(values)

	return h.q.UpdateLearningEntry(ctx, db.UpdateLearningEntryParams{
		ID: row.ID, UserID: row.UserID, TopicID: row.TopicID, Kind: f.Kind, Text: f.Text, TextHash: row.TextHash,
		Transcription: f.Transcription, Translation: f.Translation, Explanation: f.Explanation,
		NativeUsage: f.NativeUsage, Example: f.Example, Rule: f.Rule, Source: f.Source, Status: f.Status,
		UpdatedAt: db.Time(h.clock.Now()),
	})
}
