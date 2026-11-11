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

// Delete removes a teacher from the learner's list. Its running program is not
// thrown away: it is closed into the history like a reset, so what the learner
// did stays queryable, and studying the language again makes a new teacher.
func (h TeacherHandler) Delete(ctx context.Context, userID, teacherID uuid.UUID) error {
	teacher, err := h.q.GetLearningTeacher(ctx, db.GetLearningTeacherParams{ID: ids.ToPg(teacherID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return apperr.NewNotFound("teacher not found")
	}

	if err != nil {
		return err
	}

	// Close the program first: if removing the teacher then fails, asking again
	// finds nothing left to close and finishes the job.
	active, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: teacher.UserID, Language: teacher.Language})
	if err == nil {
		if _, err := h.q.ArchiveLearningPath(ctx, db.ArchiveLearningPathParams{
			ID:         active.ID,
			UserID:     active.UserID,
			Status:     db.LearningPathStatusAbandoned,
			ArchivedAt: db.Time(h.clock.Now()),
		}); err != nil {
			return err
		}
	} else if !db.IsNotFound(err) {
		return err
	}

	_, err = h.q.DeleteLearningTeacher(ctx, db.DeleteLearningTeacherParams{ID: teacher.ID, UserID: teacher.UserID})

	return err
}

// validLanguage accepts a BCP 47 tag of a sane length.
func validLanguage(code string) bool {
	if code == "" || len(code) > 35 {
		return false
	}

	_, err := language.Parse(code)

	return err == nil
}
