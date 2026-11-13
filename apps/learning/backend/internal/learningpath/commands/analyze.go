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
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
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

	if test.Status != db.LearningPathTestStatusDone {
		return db.LearningPath{}, nil, apperr.NewConflict("finish the test first")
	}

	if _, err := h.q.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: test.UserID, Language: test.Language}); err == nil {
		return db.LearningPath{}, nil, apperr.NewConflict("a program for this language is already running: finish it or reset it first")
	} else if !db.IsNotFound(err) {
		return db.LearningPath{}, nil, err
	}

	answers, err := h.q.ListLearningPathTestAnswers(ctx, test.ID)
	if err != nil {
		return db.LearningPath{}, nil, err
	}

	rows, err := h.q.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return db.LearningPath{}, nil, err
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

	prefs := cmd.Preferences.Clean()

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

		plan, err = h.planFromLibrary(ctx, cmd.UserID, learned, explain, reached, asked, correct, wrong, prefs, library, vocabulary)
		if err != nil {
			return db.LearningPath{}, nil, err
		}
	} else {
		system, prompt := analyzePrompt(learned, explain, reached, asked, correct, wrong, prefs)

		if err := h.agent.completeJSON(ctx, cmd.UserID, system, prompt, &plan, func() error {
			return validateProgram(&plan)
		}); err != nil {
			return db.LearningPath{}, nil, err
		}
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
// re-analysis after a reset, and one built at a level the learner picked
// without a test (a placement with no questions) a pick, so the history shows
// how each program came to be.
func (h AnalyzeHandler) sourceOf(ctx context.Context, test db.LearningPathTest) string {
	if questions, err := h.q.ListLearningPathTestQuestions(ctx, test.ID); err == nil && len(questions) == 0 {
		return "chosen"
	}

	history, err := h.q.ListLearningPathHistory(ctx, test.UserID)
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
