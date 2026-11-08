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
	"learning_app/backend/internal/learningpath/placement"
)

const maxTestAnswerRunes = 400

// StartTest opens a level test. A placement climbs the ladder from A1 until a
// batch fails; a final checks the program's own level. A test that is already
// open for the language is handed back instead of a second one, so a refresh
// or a double tap never splits a test in two.
type StartTest struct {
	UserID              uuid.UUID
	Kind                string // "placement" or "final"
	Language            string
	TranslationLanguage string
}

type TestHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTestHandler(q *db.Queries, clk clock.Clock) TestHandler {
	return TestHandler{q: q, clock: clk}
}

func (h TestHandler) Start(ctx context.Context, cmd StartTest) (db.LearningPathTest, error) {
	for _, code := range []string{cmd.Language, cmd.TranslationLanguage} {
		if _, err := language.Parse(code); err != nil || code == "" || len(code) > 35 {
			return db.LearningPathTest{}, apperr.NewInvalid("invalid language code")
		}
	}

	if cmd.Kind != "placement" && cmd.Kind != "final" {
		return db.LearningPathTest{}, apperr.NewInvalid("invalid test kind")
	}

	if !placement.Has(cmd.Language) {
		return db.LearningPathTest{}, apperr.NewInvalid("the level test is not available for this language yet")
	}

	open, err := h.q.GetOpenLearningPathTest(ctx, db.GetOpenLearningPathTestParams{
		UserID:   ids.ToPg(cmd.UserID),
		Language: cmd.Language,
		Kind:     db.LearningPathTestKind(cmd.Kind),
	})
	if err == nil {
		return open, nil
	}

	if !db.IsNotFound(err) {
		return db.LearningPathTest{}, err
	}

	// A placement measures toward a program, a final confirms one: it hangs off
	// the active path and asks about the path's own level.
	var path db.LearningPath
	var pathID uuid.UUID

	if cmd.Kind == "final" {
		path, err = h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: ids.ToPg(cmd.UserID), Language: cmd.Language})
		if db.IsNotFound(err) {
			return db.LearningPathTest{}, apperr.NewNotFound("no active program for this language")
		}

		if err != nil {
			return db.LearningPathTest{}, err
		}

		pathID = ids.FromPg(path.ID)
	} else {
		_, err = h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: ids.ToPg(cmd.UserID), Language: cmd.Language})
		if err == nil {
			return db.LearningPathTest{}, apperr.NewConflict("a program for this language is already running: finish it or reset it first")
		}

		if !db.IsNotFound(err) {
			return db.LearningPathTest{}, err
		}
	}

	level := string(placement.A1)
	if cmd.Kind == "final" {
		level = path.Level
	}

	return h.q.InsertLearningPathTest(ctx, db.InsertLearningPathTestParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              ids.ToPg(cmd.UserID),
		PathID:              ids.ToPgOpt(pathID),
		Kind:                db.LearningPathTestKind(cmd.Kind),
		Language:            cmd.Language,
		TranslationLanguage: cmd.TranslationLanguage,
		Level:               level,
		CreatedAt:           db.Time(h.clock.Now()),
	})
}

// TestProgress is where a test stands: how the current batch has gone and
// which questions are still to answer. A done test carries the verdict.
type TestProgress struct {
	Test      db.LearningPathTest
	Questions []placement.Question // what is still to answer on the current level
	Asked     int                  // answered on the current level
	Correct   int                  // right on the current level
	Needed    int                  // batch size of the test kind
	PassNeed  int                  // right answers that pass the level
}

// Progress reads a test and computes where it stands. It is the shared back of
// the state endpoint and of answering a batch.
func (h TestHandler) Progress(ctx context.Context, userID, testID uuid.UUID) (TestProgress, error) {
	test, err := h.q.GetLearningPathTest(ctx, db.GetLearningPathTestParams{ID: ids.ToPg(testID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return TestProgress{}, apperr.NewNotFound("test not found")
	}

	if err != nil {
		return TestProgress{}, err
	}

	answers, err := h.q.ListLearningPathTestAnswers(ctx, ids.ToPg(testID))
	if err != nil {
		return TestProgress{}, err
	}

	return progressOf(test, answers), nil
}

// progressOf folds the stored answers into where the test stands. Answers of
// passed levels stay in the table as history; only the current level counts.
func progressOf(test db.LearningPathTest, answers []db.LearningPathTestAnswer) TestProgress {
	p := TestProgress{Test: test, Needed: placement.BatchNeed(string(test.Kind)), PassNeed: placement.PassNeed(string(test.Kind))}

	answered := map[string]bool{}

	for _, a := range answers {
		_, level, ok := placement.Find(test.Language, a.QuestionKey)
		if !ok || level != placement.Level(test.Level) {
			continue
		}

		answered[a.QuestionKey] = true
		p.Asked++

		if a.Correct {
			p.Correct++
		}
	}

	if test.Status == db.LearningPathTestStatusInProgress {
		exclude := make(map[string]bool, len(answers))
		for _, a := range answers {
			exclude[a.QuestionKey] = true
		}

		p.Questions = placement.Batch(test.Language, placement.Level(test.Level), exclude, p.Needed-p.Asked)
	}

	return p
}

// Answer is one answered question of a batch.
type Answer struct {
	Key    string
	Answer string
}

// AnswerBatch records a batch of answers and moves the ladder. Every key must
// belong to the level the test is asking about now; re-sending a key is
// ignored, so a retry is safe. When the level's batch is complete the test
// either climbs or finishes, and a finished final test the learner passed
// finishes the program with it.
func (h TestHandler) Answer(ctx context.Context, userID, testID uuid.UUID, given []Answer) (TestProgress, error) {
	test, err := h.q.GetLearningPathTest(ctx, db.GetLearningPathTestParams{ID: ids.ToPg(testID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return TestProgress{}, apperr.NewNotFound("test not found")
	}

	if err != nil {
		return TestProgress{}, err
	}

	if test.Status != db.LearningPathTestStatusInProgress {
		return TestProgress{}, apperr.NewConflict("this test is already done")
	}

	if len(given) == 0 || len(given) > placement.BatchNeed(string(test.Kind)) {
		return TestProgress{}, apperr.NewInvalid("a batch of 1 to 8 answers")
	}

	for _, a := range given {
		if strings.TrimSpace(a.Key) == "" || a.Key != strings.TrimSpace(a.Key) {
			return TestProgress{}, apperr.NewInvalid("invalid question key")
		}

		if utf8.RuneCountInString(a.Answer) > maxTestAnswerRunes {
			return TestProgress{}, apperr.NewInvalid("an answer is too long")
		}
	}

	for _, a := range given {
		q, level, ok := placement.Find(test.Language, a.Key)
		if !ok || level != placement.Level(test.Level) {
			return TestProgress{}, apperr.NewInvalid("unknown question for this level: " + a.Key)
		}

		if _, err := h.q.InsertLearningPathTestAnswer(ctx, db.InsertLearningPathTestAnswerParams{
			TestID:      test.ID,
			QuestionKey: a.Key,
			Answer:      strings.TrimSpace(a.Answer),
			Correct:     placement.Grade(q, a.Answer),
		}); err != nil {
			return TestProgress{}, err
		}
	}

	p, err := h.Progress(ctx, userID, testID)
	if err != nil {
		return p, err
	}

	if p.Asked < p.Needed {
		return p, nil
	}

	now := db.Time(h.clock.Now())
	passed := p.Correct >= p.PassNeed

	// finish marks the test done in the store and on the value we hand back.
	finish := func(reached string, ok bool) error {
		if err := h.q.FinishLearningPathTest(ctx, db.FinishLearningPathTestParams{ID: test.ID, UserID: test.UserID, LevelReached: reached, Passed: ok, CompletedAt: now}); err != nil {
			return err
		}

		p.Test.Status = db.LearningPathTestStatusDone
		p.Test.LevelReached = reached
		p.Test.Passed = ok
		p.Test.CompletedAt = now
		p.Questions = nil

		return nil
	}

	if test.Kind == db.LearningPathTestKindFinal {
		if err := finish(test.Level, passed); err != nil {
			return p, err
		}

		if passed && test.PathID.Valid {
			if _, err := h.q.ArchiveLearningPath(ctx, db.ArchiveLearningPathParams{
				ID:         test.PathID,
				UserID:     test.UserID,
				Status:     db.LearningPathStatusFinished,
				ArchivedAt: now,
			}); err != nil {
				return p, err
			}
		}

		return p, nil
	}

	// Placement: a pass climbs, a fail lands on the level under the one that
	// stopped the climb. Passing the top rung is finishing the ladder.
	if passed {
		if next, ok := placement.Next(placement.Level(test.Level)); ok {
			if err := h.q.AdvanceLearningPathTest(ctx, db.AdvanceLearningPathTestParams{ID: test.ID, UserID: test.UserID, Level: string(next)}); err != nil {
				return p, err
			}

			p.Test.Level = string(next)
			p.Questions = placement.Batch(test.Language, next, nil, p.Needed)
			p.Asked, p.Correct = 0, 0

			return p, nil
		}

		return p, finish(test.Level, true)
	}

	return p, finish(string(placement.Below(placement.Level(test.Level))), false)
}
