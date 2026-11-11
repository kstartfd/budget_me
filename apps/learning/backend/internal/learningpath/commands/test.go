package commands

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/language"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/learningpath/placement"
)

const maxTestAnswerRunes = 400

// StartTest opens a level test. A placement climbs the ladder from A1 until the
// examiner stops it; a final checks the program's own level. A test that is
// already open for the language is handed back instead of a second one, so a
// refresh or a double tap never splits a test in two.
//
// The questions are written for the learner by their own agent, and the first
// batch is written before the test exists: a learner whose agent cannot be
// reached never gets a test that could not be taken.
type StartTest struct {
	UserID              uuid.UUID
	Kind                string // "placement" or "final"
	Language            string
	TranslationLanguage string
	// StartLevel is where a placement begins when it is not at A1: the rung
	// after the level a final test has just confirmed. Empty means A1.
	StartLevel string
}

type TestHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
	agent Agent
}

func NewTestHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, agent Agent) TestHandler {
	return TestHandler{pool: pool, q: q, clock: clk, agent: agent}
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

	// A teacher fixes the pair of its language: whatever the request says, the
	// rules stay in the language the teacher was created with.
	teacher, err := h.q.GetLearningTeacherByLanguage(ctx, db.GetLearningTeacherByLanguageParams{UserID: ids.ToPg(cmd.UserID), Language: cmd.Language})
	if err == nil {
		cmd.TranslationLanguage = teacher.TranslationLanguage
	} else if !db.IsNotFound(err) {
		return db.LearningPathTest{}, err
	}

	open, err := h.q.GetOpenLearningPathTest(ctx, db.GetOpenLearningPathTestParams{
		UserID:   ids.ToPg(cmd.UserID),
		Language: cmd.Language,
		Kind:     db.LearningPathTestKind(cmd.Kind),
	})
	if err == nil {
		return open, h.ensureQuestions(ctx, open)
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
	} else if cmd.StartLevel != "" {
		level, err = h.nextLevel(ctx, cmd)
		if err != nil {
			return db.LearningPathTest{}, err
		}
	}

	// The first batch doubles as the check that the agent can be reached: no
	// agent, a rejected key or an unreadable reply stops the test right here.
	questions, err := h.generate(ctx, cmd.UserID, cmd.Kind, cmd.Language, cmd.TranslationLanguage, placement.Level(level), nil)
	if err != nil {
		return db.LearningPathTest{}, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return db.LearningPathTest{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)

	// Studying a language means having its teacher.
	if _, err := tq.InsertLearningTeacher(ctx, db.InsertLearningTeacherParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              ids.ToPg(cmd.UserID),
		Language:            cmd.Language,
		TranslationLanguage: cmd.TranslationLanguage,
		CreatedAt:           db.Time(h.clock.Now()),
	}); err != nil {
		return db.LearningPathTest{}, err
	}

	test, err := tq.InsertLearningPathTest(ctx, db.InsertLearningPathTestParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              ids.ToPg(cmd.UserID),
		PathID:              ids.ToPgOpt(pathID),
		Kind:                db.LearningPathTestKind(cmd.Kind),
		Language:            cmd.Language,
		TranslationLanguage: cmd.TranslationLanguage,
		Level:               level,
		CreatedAt:           db.Time(h.clock.Now()),
	})
	if err != nil {
		return db.LearningPathTest{}, err
	}

	if err := storeQuestions(ctx, tq, test.ID, placement.Level(level), questions); err != nil {
		return db.LearningPathTest{}, err
	}

	return test, tx.Commit(ctx)
}

// nextLevel checks a placement that starts above A1: the learner must have
// confirmed a level in a final test, and the start can only be the rung right
// above it, so the climb carries on from where they proved themselves.
func (h TestHandler) nextLevel(ctx context.Context, cmd StartTest) (string, error) {
	history, err := h.q.ListLearningPathHistory(ctx, ids.ToPg(cmd.UserID))
	if err != nil {
		return "", err
	}

	for _, path := range history {
		if path.Language != cmd.Language || path.Status != db.LearningPathStatusFinished {
			continue
		}

		next, ok := placement.Next(placement.Level(path.Level))
		if ok && string(next) == cmd.StartLevel {
			return cmd.StartLevel, nil
		}

		break // only the newest finished program counts
	}

	return "", apperr.NewInvalid("a test can start above A1 only at the level after the one you confirmed")
}

// generate has the learner's agent write one batch of a level. Prompts already
// asked in the test travel with the request so the agent does not repeat them.
func (h TestHandler) generate(ctx context.Context, userID uuid.UUID, kind, lang, translation string, level placement.Level, asked []string) ([]placement.Question, error) {
	count := placement.BatchNeed(kind)
	learned := learningcommands.LanguageName(lang)
	native := learningcommands.LanguageName(translation)
	system, prompt := testQuestionsPrompt(learned, native, kind, level, count, asked)

	var batch generatedBatch
	if err := h.agent.completeJSON(ctx, userID, system, prompt, &batch, func() error {
		return validateBatch(&batch, count)
	}); err != nil {
		return nil, err
	}

	return questionsOf(batch, level), nil
}

// ensureQuestions writes the batch of an open test that has none for the level
// it is on, which is how a test opened before questions were stored resumes.
func (h TestHandler) ensureQuestions(ctx context.Context, test db.LearningPathTest) error {
	rows, err := h.q.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return err
	}

	var asked []string

	for _, r := range rows {
		if r.Level == test.Level {
			return nil
		}

		asked = append(asked, r.Prompt)
	}

	questions, err := h.generate(ctx, ids.FromPg(test.UserID), string(test.Kind), test.Language, test.TranslationLanguage, placement.Level(test.Level), asked)
	if err != nil {
		return err
	}

	return storeQuestions(ctx, h.q, test.ID, placement.Level(test.Level), questions)
}

// storeQuestions keeps a batch with its test, answer keys included.
func storeQuestions(ctx context.Context, q *db.Queries, testID pgtype.UUID, level placement.Level, questions []placement.Question) error {
	for i, question := range questions {
		options, err := json.Marshal(question.Options)
		if err != nil {
			return err
		}

		if err := q.InsertLearningPathTestQuestion(ctx, db.InsertLearningPathTestQuestionParams{
			TestID:      testID,
			QuestionKey: question.Key,
			Level:       string(level),
			Position:    int32(i),
			Skill:       string(question.Skill),
			Text:        question.Text,
			Prompt:      question.Prompt,
			Options:     options,
			Answer:      question.Answer,
		}); err != nil {
			return err
		}
	}

	return nil
}

// questionOf reads a stored question back.
func questionOf(row db.LearningPathTestQuestion) placement.Question {
	var options []string
	if len(row.Options) > 0 {
		_ = json.Unmarshal(row.Options, &options)
	}

	return placement.Question{
		Key:     row.QuestionKey,
		Skill:   placement.Skill(row.Skill),
		Text:    row.Text,
		Prompt:  row.Prompt,
		Options: options,
		Answer:  row.Answer,
	}
}

// TestProgress is where a test stands: how the current batch has gone and
// which questions are still to answer. A done test carries the verdict.
type TestProgress struct {
	Test      db.LearningPathTest
	Questions []placement.Question // what is still to answer on the current level
	Asked     int                  // answered on the current level
	Correct   int                  // right on the current level
	Needed    int                  // questions of the current level's batch
	// StartLevel is the rung the test began at, so a screen can say how many
	// pages (one per rung) a test can have at most.
	StartLevel string
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

	answers, questions, err := h.stored(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}

	return progressOf(test, answers, questions), nil
}

func (h TestHandler) stored(ctx context.Context, testID pgtype.UUID) ([]db.LearningPathTestAnswer, []db.LearningPathTestQuestion, error) {
	answers, err := h.q.ListLearningPathTestAnswers(ctx, testID)
	if err != nil {
		return nil, nil, err
	}

	questions, err := h.q.ListLearningPathTestQuestions(ctx, testID)
	if err != nil {
		return nil, nil, err
	}

	return answers, questions, nil
}

// progressOf folds the stored questions and answers into where the test
// stands. Answers of passed levels stay in the table as history; only the
// current level counts.
func progressOf(test db.LearningPathTest, answers []db.LearningPathTestAnswer, questions []db.LearningPathTestQuestion) TestProgress {
	p := TestProgress{Test: test, Needed: placement.BatchNeed(string(test.Kind)), StartLevel: test.Level}

	for _, r := range questions {
		if placement.Index(placement.Level(r.Level)) < placement.Index(placement.Level(p.StartLevel)) {
			p.StartLevel = r.Level
		}
	}

	answered := map[string]bool{}
	for _, a := range answers {
		answered[a.QuestionKey] = true
	}

	var level []db.LearningPathTestQuestion

	for _, r := range questions {
		if r.Level == test.Level {
			level = append(level, r)
		}
	}

	if len(level) > 0 {
		p.Needed = len(level)
	}

	correct := map[string]bool{}
	for _, a := range answers {
		correct[a.QuestionKey] = a.Correct
	}

	for _, r := range level {
		if answered[r.QuestionKey] {
			p.Asked++

			if correct[r.QuestionKey] {
				p.Correct++
			}

			continue
		}

		if test.Status == db.LearningPathTestStatusInProgress {
			p.Questions = append(p.Questions, questionOf(r))
		}
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
// ignored, so a retry is safe. When the level's batch is complete the examiner
// (the learner's agent) decides: the test climbs, or it finishes with the level
// the learner is placed at. A finished final test the learner passed finishes
// the program with it.
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
		return TestProgress{}, apperr.NewInvalid("a batch of 1 to 10 answers")
	}

	for _, a := range given {
		if strings.TrimSpace(a.Key) == "" || a.Key != strings.TrimSpace(a.Key) {
			return TestProgress{}, apperr.NewInvalid("invalid question key")
		}

		if utf8.RuneCountInString(a.Answer) > maxTestAnswerRunes {
			return TestProgress{}, apperr.NewInvalid("an answer is too long")
		}
	}

	_, rows, err := h.stored(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}

	onLevel := map[string]placement.Question{}

	for _, r := range rows {
		if r.Level == test.Level {
			onLevel[r.QuestionKey] = questionOf(r)
		}
	}

	for _, a := range given {
		q, ok := onLevel[a.Key]
		if !ok {
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

	answers, rows, err := h.stored(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}

	p := progressOf(test, answers, rows)
	if p.Asked < p.Needed {
		return p, nil
	}

	level := placement.Level(test.Level)

	call, err := h.judge(ctx, test, level, p, answers, rows)
	if err != nil {
		return p, err
	}

	now := db.Time(h.clock.Now())
	climbs := call.Verdict == "climb"

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
		if err := finish(test.Level, climbs); err != nil {
			return p, err
		}

		if climbs && test.PathID.Valid {
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

	// Placement: a climb goes up a rung, a stop lands where the examiner placed
	// the learner. Climbing the top rung is finishing the ladder.
	if !climbs {
		return p, finish(call.Level, false)
	}

	next, ok := placement.Next(level)
	if !ok {
		return p, finish(test.Level, true)
	}

	asked := make([]string, len(rows))
	for i, r := range rows {
		asked[i] = r.Prompt
	}

	questions, err := h.generate(ctx, ids.FromPg(test.UserID), string(test.Kind), test.Language, test.TranslationLanguage, next, asked)
	if err != nil {
		return p, err
	}

	if err := h.advance(ctx, test, next, questions); err != nil {
		return p, err
	}

	p.Test.Level = string(next)
	p.Questions = questions
	p.Needed = len(questions)
	p.Asked, p.Correct = 0, 0

	return p, nil
}

// judge has the examiner read the finished batch of the level.
func (h TestHandler) judge(ctx context.Context, test db.LearningPathTest, level placement.Level, p TestProgress, answers []db.LearningPathTestAnswer, rows []db.LearningPathTestQuestion) (verdict, error) {
	chosen := map[string]db.LearningPathTestAnswer{}
	for _, a := range answers {
		chosen[a.QuestionKey] = a
	}

	var batch []judged

	for _, r := range rows {
		if a, ok := chosen[r.QuestionKey]; ok && r.Level == test.Level {
			batch = append(batch, judged{question: questionOf(r), chosen: a.Answer, correct: a.Correct})
		}
	}

	learned := learningcommands.LanguageName(test.Language)
	native := learningcommands.LanguageName(test.TranslationLanguage)
	system, prompt := testJudgePrompt(learned, native, string(test.Kind), level, batch)

	var v verdict
	if err := h.agent.completeJSON(ctx, ids.FromPg(test.UserID), system, prompt, &v, func() error {
		return validateVerdict(&v, level, p.Asked, p.Correct)
	}); err != nil {
		return verdict{}, err
	}

	return v, nil
}

// advance moves the test up a rung together with the batch written for it.
func (h TestHandler) advance(ctx context.Context, test db.LearningPathTest, next placement.Level, questions []placement.Question) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)

	if err := tq.AdvanceLearningPathTest(ctx, db.AdvanceLearningPathTestParams{ID: test.ID, UserID: test.UserID, Level: string(next)}); err != nil {
		return err
	}

	if err := storeQuestions(ctx, tq, test.ID, next, questions); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
