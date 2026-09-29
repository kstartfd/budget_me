package commands

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type EntryFields struct {
	TopicID       uuid.UUID              `json:"topicId"`
	Kind          db.LearningEntryKind   `json:"kind"`
	Text          string                 `json:"text"`
	Transcription string                 `json:"transcription"`
	Translation   string                 `json:"translation"`
	NativeUsage   string                 `json:"nativeUsage"`
	Example       string                 `json:"example"`
	Rule          string                 `json:"rule"`
	Status        db.LearningEntryStatus `json:"status"`
}

func (f EntryFields) Normalize() (EntryFields, error) {
	f.Text = strings.TrimSpace(f.Text)
	f.Transcription = strings.TrimSpace(f.Transcription)
	f.Translation = strings.TrimSpace(f.Translation)
	f.NativeUsage = strings.TrimSpace(f.NativeUsage)
	f.Example = strings.TrimSpace(f.Example)

	f.Rule = strings.TrimSpace(f.Rule)
	if f.Kind == "" {
		f.Kind = db.LearningEntryKindPhrase
	}

	if f.Status == "" {
		f.Status = db.LearningEntryStatusInbox
	}

	if f.TopicID == uuid.Nil || f.Text == "" || utf8.RuneCountInString(f.Text) > 1000 {
		return f, apperr.NewInvalid("topicId and text (1..1000 characters) are required")
	}

	if f.Kind != db.LearningEntryKindWord && f.Kind != db.LearningEntryKindPhrase {
		return f, apperr.NewInvalid("invalid entry kind")
	}

	if f.Status != db.LearningEntryStatusInbox && f.Status != db.LearningEntryStatusLearning && f.Status != db.LearningEntryStatusKnown {
		return f, apperr.NewInvalid("invalid entry status")
	}

	for _, s := range []string{f.Transcription, f.Translation, f.NativeUsage, f.Example, f.Rule} {
		if utf8.RuneCountInString(s) > 4000 {
			return f, apperr.NewInvalid("entry field exceeds 4000 characters")
		}
	}

	return f, nil
}

type SaveEntry struct {
	UserID uuid.UUID
	ID     uuid.UUID
	Fields EntryFields
}
type EntryHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewEntryHandler(q *db.Queries, clk clock.Clock) EntryHandler {
	return EntryHandler{q: q, clock: clk}
}

func (h EntryHandler) Handle(ctx context.Context, cmd SaveEntry) (db.LearningEntry, error) {
	f, err := cmd.Fields.Normalize()
	if err != nil {
		return db.LearningEntry{}, err
	}

	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(f.TopicID), UserID: ids.ToPg(cmd.UserID)}); err != nil {
		if db.IsNotFound(err) {
			return db.LearningEntry{}, apperr.NewNotFound("topic not found")
		}

		return db.LearningEntry{}, err
	}

	now := db.Time(h.clock.Now())
	if cmd.ID == uuid.Nil {
		return h.q.InsertLearningEntry(ctx, db.InsertLearningEntryParams{ID: ids.ToPg(ids.New()), UserID: ids.ToPg(cmd.UserID), TopicID: ids.ToPg(f.TopicID), Kind: f.Kind, Text: f.Text, Transcription: f.Transcription, Translation: f.Translation, NativeUsage: f.NativeUsage, Example: f.Example, Rule: f.Rule, Status: f.Status, CreatedAt: now, UpdatedAt: now})
	}

	row, err := h.q.UpdateLearningEntry(ctx, db.UpdateLearningEntryParams{ID: ids.ToPg(cmd.ID), UserID: ids.ToPg(cmd.UserID), TopicID: ids.ToPg(f.TopicID), Kind: f.Kind, Text: f.Text, Transcription: f.Transcription, Translation: f.Translation, NativeUsage: f.NativeUsage, Example: f.Example, Rule: f.Rule, Status: f.Status, UpdatedAt: now})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	return row, err
}

func (h EntryHandler) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := h.q.DeleteLearningEntry(ctx, db.DeleteLearningEntryParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID)})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewNotFound("entry not found")
	}

	return nil
}
