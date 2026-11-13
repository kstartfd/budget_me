package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

// ExplainLanguage is the language a teacher's words are written in: the rules,
// the program, the exercises and the explanation of a mistake. It is the
// learner's own language unless they asked for the learned one.
func ExplainLanguage(t db.LearningTeacher) string {
	if t.ExplainInTarget {
		return t.Language
	}

	return t.TranslationLanguage
}

// explanationOf is the language the teacher of a language explains in, for the
// jobs that only know a program or a test. A learner whose teacher is gone (its
// program is in the history) is explained to in the pair the program was made
// with.
func explanationOf(ctx context.Context, q *db.Queries, userID pgtype.UUID, language, translation string) string {
	teacher, err := q.GetLearningTeacherByLanguage(ctx, db.GetLearningTeacherByLanguageParams{UserID: userID, Language: language})
	if err != nil {
		return translation
	}

	return ExplainLanguage(teacher)
}

// SetExplain switches a teacher between explaining in the learner's own
// language and in the learned one. It changes what is written from now on and
// which rules are shown. What was written before stays as it is, except what
// the learner has not started: the exercise sets nobody answered are dropped
// and written again, in the new language, when their topics are opened.
func (h TeacherHandler) SetExplain(ctx context.Context, userID, teacherID uuid.UUID, inTarget bool) (db.LearningTeacher, error) {
	before, err := h.q.GetLearningTeacher(ctx, db.GetLearningTeacherParams{ID: ids.ToPg(teacherID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return db.LearningTeacher{}, apperr.NewNotFound("teacher not found")
	}

	if err != nil {
		return db.LearningTeacher{}, err
	}

	if before.ExplainInTarget == inTarget {
		return before, nil
	}

	teacher, err := h.q.SetLearningTeacherExplain(ctx, db.SetLearningTeacherExplainParams{ID: before.ID, UserID: before.UserID, ExplainInTarget: inTarget})
	if err != nil {
		return db.LearningTeacher{}, err
	}

	// A practice sitting on a rule is written in the old language: it is a
	// sitting, not a record, so it goes rather than stay half in each.
	if err := h.q.DeleteLearningRulePracticeOfLanguage(ctx, db.DeleteLearningRulePracticeOfLanguageParams{UserID: teacher.UserID, Language: teacher.Language}); err != nil {
		return db.LearningTeacher{}, err
	}

	if err := h.q.DeleteUntouchedLearningPathExercises(ctx, db.DeleteUntouchedLearningPathExercisesParams{UserID: teacher.UserID, Language: teacher.Language}); err != nil {
		return db.LearningTeacher{}, err
	}

	return teacher, nil
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
