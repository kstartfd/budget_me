package commands

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/language"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type SaveTopic struct {
	UserID              uuid.UUID
	ID                  uuid.UUID
	Name                string
	Language            string
	TranslationLanguage string
}

type TopicHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTopicHandler(q *db.Queries, clk clock.Clock) TopicHandler {
	return TopicHandler{q: q, clock: clk}
}

func (h TopicHandler) Handle(ctx context.Context, cmd SaveTopic) (db.LearningTopic, error) {
	cmd.Name = strings.TrimSpace(cmd.Name)
	if cmd.Name == "" || utf8.RuneCountInString(cmd.Name) > 80 {
		return db.LearningTopic{}, apperr.NewInvalid("topic name must be 1..80 characters")
	}

	for _, code := range []string{cmd.Language, cmd.TranslationLanguage} {
		if _, err := language.Parse(code); err != nil || code == "" || len(code) > 35 {
			return db.LearningTopic{}, apperr.NewInvalid("invalid language code")
		}
	}

	if cmd.ID == uuid.Nil {
		return h.q.InsertLearningTopic(ctx, db.InsertLearningTopicParams{ID: ids.ToPg(ids.New()), UserID: ids.ToPg(cmd.UserID), Name: cmd.Name, Language: cmd.Language, TranslationLanguage: cmd.TranslationLanguage, CreatedAt: db.Time(h.clock.Now())})
	}

	row, err := h.q.UpdateLearningTopic(ctx, db.UpdateLearningTopicParams{ID: ids.ToPg(cmd.ID), UserID: ids.ToPg(cmd.UserID), Name: cmd.Name, Language: cmd.Language, TranslationLanguage: cmd.TranslationLanguage})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("topic not found")
	}

	return row, err
}

func (h TopicHandler) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := h.q.DeleteLearningTopic(ctx, db.DeleteLearningTopicParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID)})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewNotFound("topic not found")
	}

	return nil
}
