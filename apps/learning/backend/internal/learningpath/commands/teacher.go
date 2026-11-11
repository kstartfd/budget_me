package commands

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/text/language"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// CreateTeacher adds a language to study, with the language its rules are
// explained in. There is one teacher per language: asking for a language that
// already has one hands the existing teacher back, with its own pair, so a
// double tap or a second device never splits a language in two.
type CreateTeacher struct {
	UserID              uuid.UUID
	Language            string
	TranslationLanguage string
}

type TeacherHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTeacherHandler(q *db.Queries, clk clock.Clock) TeacherHandler {
	return TeacherHandler{q: q, clock: clk}
}

func (h TeacherHandler) Create(ctx context.Context, cmd CreateTeacher) (db.LearningTeacher, error) {
	if !validLanguage(cmd.Language) || !validLanguage(cmd.TranslationLanguage) {
		return db.LearningTeacher{}, apperr.NewInvalid("invalid language code")
	}

	if cmd.Language == cmd.TranslationLanguage {
		return db.LearningTeacher{}, apperr.NewInvalid("the language to learn and the language of the rules must differ")
	}

	return h.q.InsertLearningTeacher(ctx, db.InsertLearningTeacherParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              ids.ToPg(cmd.UserID),
		Language:            cmd.Language,
		TranslationLanguage: cmd.TranslationLanguage,
		CreatedAt:           db.Time(h.clock.Now()),
	})
}

// validLanguage accepts a BCP 47 tag of a sane length.
func validLanguage(code string) bool {
	if code == "" || len(code) > 35 {
		return false
	}

	_, err := language.Parse(code)

	return err == nil
}
