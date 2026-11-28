package commands

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
	"learning_app/backend/internal/llm"
)

// Analyze reads a finished placement test and has the learner's agent turn it
// into a program: the level, a summary of what the test showed, and the
// ordered topics. The program is saved before it is returned; a re-analysis
// after a reset keeps the old one in the history, since archiving never
// deletes.
type Analyze struct {
	UserID      uuid.UUID
	TestID      uuid.UUID
	Preferences Preferences
}

type AnalyzeHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
	agent Agent
}

func NewAnalyzeHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, agent Agent) AnalyzeHandler {
	return AnalyzeHandler{pool: pool, q: q, clock: clk, agent: agent}
}

func (h AnalyzeHandler) Handle(ctx context.Context, cmd Analyze) (db.LearningPath, []db.LearningPathTopic, error) {
	test, err := h.q.GetLearningPathTest(ctx, db.GetLearningPathTestParams{ID: ids.ToPg(cmd.TestID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return db.LearningPath{}, nil, apperr.NewNotFound("test not found")
	}

	if err != nil {
		return db.LearningPath{}, nil, err
	}

	if test.Kind != db.LearningPathTestKindPlacement {
		return db.LearningPath{}, nil, apperr.NewInvalid("only a placement test builds a program")
	}

	if !canBuildFromTest(test) {
		return db.LearningPath{}, nil, apperr.NewConflict("finish the test first")
	}

	if _, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: test.UserID, Language: test.Language}); err == nil {
		return db.LearningPath{}, nil, apperr.NewConflict("a program for this language is already running: finish it or reset it first")
	} else if !db.IsNotFound(err) {
		return db.LearningPath{}, nil, err
	}

	prefs := cmd.Preferences.Clean()
	var plan program
	if test.AssessmentType.Valid {
		plan, err = analyzeScreening(test, prefs)
	} else {
		plan, err = h.legacyPlan(ctx, cmd.UserID, test, prefs)
	}
	if err != nil {
		return db.LearningPath{}, nil, err
	}

	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return db.LearningPath{}, nil, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return db.LearningPath{}, nil, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)
	if test.AssessmentType.Valid {
		if _, err := lockAssessment(ctx, tq, test.UserID, test.ID); err != nil {
			return db.LearningPath{}, nil, err
		}
	}
	now := h.clock.Now()

	path, err := tq.InsertLearningPath(ctx, db.InsertLearningPathParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              test.UserID,
		Language:            test.Language,
		TranslationLanguage: test.TranslationLanguage,
		Level:               test.LevelReached,
		Summary:             plan.Summary,
		Source:              h.sourceOf(ctx, test),
		PlacementTestID:     test.ID,
		Preferences:         prefsJSON,
		CreatedAt:           db.Time(now),
	})
	if err != nil {
		return db.LearningPath{}, nil, err
	}

	topics := make([]db.LearningPathTopic, len(plan.Topics))
	for i, t := range plan.Topics {
		vocabulary, err := json.Marshal(t.Vocabulary)
		if err != nil {
			return db.LearningPath{}, nil, err
		}

		topics[i], err = tq.InsertLearningPathTopic(ctx, db.InsertLearningPathTopicParams{
			ID:         ids.ToPg(ids.New()),
			PathID:     path.ID,
			Position:   int32(i),
			Skill:      t.Skill,
			Title:      t.Title,
			Goal:       t.Goal,
			Unit:       t.Unit,
			Theory:     t.Theory,
			Vocabulary: vocabulary,
			CreatedAt:  db.Time(now),
			SourceID:   t.SourceID,
		})
		if err != nil {
			return db.LearningPath{}, nil, err
		}
	}

	return path, topics, tx.Commit(ctx)
}

// sourceOf calls the first program of a language a placement, every later one a
// re-analysis after a reset, one built at a level the learner had reached
// without a test (a placement with no questions) a pick, and one built at a
// level they only claimed a self-reported one, so the history shows how each
// program came to be.
func (h AnalyzeHandler) sourceOf(ctx context.Context, test db.LearningPathTest) string {
	if test.AssessmentType.Valid {
		return assessmentPathSource
	}

	history, err := h.q.ListLearningPathHistory(ctx, test.UserID)

	if questions, qerr := h.q.ListLearningPathTestQuestions(ctx, test.ID); qerr == nil && len(questions) == 0 {
		// No questions: a pick of a proven level, unless the history does not reach it.
		if err == nil {
			if reached, ok := ReachedLevel(history, test.Language); !ok || placement.Index(placement.Level(test.LevelReached)) > placement.Index(reached) {
				return SelfReportedSource
			}
		}

		return "chosen"
	}

	if err != nil {
		return "placement"
	}

	for _, path := range history {
		if path.Language == test.Language {
			return "reset"
		}
	}

	return "placement"
}

func (h AnalyzeHandler) legacyPlan(ctx context.Context, userID uuid.UUID, test db.LearningPathTest, prefs Preferences) (program, error) {
	answers, err := h.q.ListLearningPathTestAnswers(ctx, test.ID)
	if err != nil {
		return program{}, err
	}

	rows, err := h.q.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return program{}, err
	}

	written := make(map[string]db.LearningPathTestQuestion, len(rows))
	for _, r := range rows {
		written[r.QuestionKey] = r
	}

	var asked, correct int
	var wrong []wrongAnswer

	for _, a := range answers {
		r, ok := written[a.QuestionKey]
		if !ok {
			continue
		}

		asked++

		if a.Correct {
			correct++
		} else {
			wrong = append(wrong, wrongAnswer{question: questionOf(r), level: placement.Level(r.Level), chosen: a.Answer})
		}
	}

	// The program is written in the language the learner's teacher explains in:
	// their own, or the one they learn.
	explainCode := explanationOf(ctx, h.q, test.UserID, test.Language, test.TranslationLanguage)
	learned := learningcommands.LanguageName(test.Language)
	explain := learningcommands.LanguageName(explainCode)

	reached := placement.Level(test.LevelReached)

	var plan program

	// A pair with a library is planned from it, so the rules and words a
	// learner reads were written by people; otherwise the agent writes it all.
	if library := libraryFor(test.Language, explainCode, reached); len(library) > 0 {
		// Key words stay translated into the learner's own language whatever
		// the explanations are written in.
		vocabulary := func(id string) []curriculum.Word {
			own, _ := curriculum.Find(test.Language, test.TranslationLanguage, id)

			return own.Vocabulary
		}

		plan, err = h.planFromLibrary(ctx, userID, learned, explain, reached, asked, correct, wrong, prefs, library, vocabulary)
		if err != nil {
			return program{}, err
		}
	} else {
		system, prompt := analyzePrompt(learned, explain, reached, asked, correct, wrong, prefs)

		if err := h.agent.completeJSON(ctx, userID, llm.PurposeProgram, system, prompt, &plan, func() error {
			return validateProgram(&plan)
		}); err != nil {
			return program{}, err
		}
	}

	return plan, nil
}

func canBuildFromTest(test db.LearningPathTest) bool {
	if test.Status != db.LearningPathTestStatusDone {
		return false
	}
	return !test.AssessmentType.Valid || test.AssessmentStatus.String == assessment.Completed ||
		test.AssessmentStatus.String == assessment.Partial
}
