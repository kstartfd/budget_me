package commands

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
)

// A sitting on a rule lives while the learner is on its page. Leaving it only
// shortens its life to practiceGrace, so a reload, a swipe away or a switch to
// another app does not throw away what the agent was paid to write; coming back
// lifts that. A sitting nobody leaves (the phone killed the app) still ends
// after practiceIdle.
const (
	practiceIdle  = 6 * time.Hour
	practiceGrace = 10 * time.Minute

	maxAnswerRunes = 300
)

// RuleSubject is the rule of the rule book a sitting is about, with the
// languages it is explained in. The controller fills it from the library, so
// no one gets a rule they have no teacher for. Explanation is the language the
// teacher explains in (the learner's own, or the learned one) and is what the
// sitting keeps; NativeName is the learner's own language, which the example
// sentences are translated into either way.
type RuleSubject struct {
	ID              string
	Language        string
	Explanation     string
	LanguageName    string
	ExplanationName string
	NativeName      string
	Level           string
	Title           string
	Theory          string
	Forms           []string
}

// PracticeExample is a sentence an agent wrote to show the rule.
type PracticeExample struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

// PracticeItem is one line of a drill. Answers and Note are the key: they stay
// on the server until the learner checks the line or asks for the key.
type PracticeItem struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Hint    string   `json:"hint"`
	Options []string `json:"options"`
	Answers []string `json:"answers"`
	Note    string   `json:"note"`
}

// PracticeBlock is one drill: an instruction and its lines, all of one kind.
type PracticeBlock struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Instruction string         `json:"instruction"`
	Items       []PracticeItem `json:"items"`
}

// Practice is the sitting of a learner on one rule. Without one, ID is nil and
// both lists are empty.
type Practice struct {
	ID       uuid.UUID
	Examples []PracticeExample
	Blocks   []PracticeBlock
}

// Check is what the learner is told about a line: whether the answer was
// right, and the key with the reason.
type Check struct {
	Correct bool
	Key     string
	Note    string
}

type PracticeHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
	agent Agent
}

func NewPracticeHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, agent Agent) PracticeHandler {
	return PracticeHandler{pool: pool, q: q, clock: clk, agent: agent}
}

func practiceOf(row db.LearningRulePractice) (Practice, error) {
	p := Practice{ID: ids.FromPg(row.ID), Examples: []PracticeExample{}, Blocks: []PracticeBlock{}}

	if err := json.Unmarshal(row.Examples, &p.Examples); err != nil {
		return Practice{}, err
	}

	if err := json.Unmarshal(row.Exercises, &p.Blocks); err != nil {
		return Practice{}, err
	}

	return p, nil
}

// sweep drops the learner's sittings whose time has run out, with their chats.
func (h PracticeHandler) sweep(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return h.q.DeleteExpiredLearningRulePractice(ctx, db.DeleteExpiredLearningRulePracticeParams{UserID: ids.ToPg(userID), ExpiresAt: db.Time(now)})
}

// Open is the learner arriving on the page of a rule: what is left of the
// sitting comes back and its life is long again.
func (h PracticeHandler) Open(ctx context.Context, userID uuid.UUID, ruleID string) (Practice, error) {
	now := h.clock.Now()

	if err := h.sweep(ctx, userID, now); err != nil {
		return Practice{}, err
	}

	row, err := h.q.TouchLearningRulePractice(ctx, db.TouchLearningRulePracticeParams{
		UserID: ids.ToPg(userID), RuleID: ruleID, ExpiresAt: db.Time(now.Add(practiceIdle)), ExpiresAt_2: db.Time(now),
	})
	if db.IsNotFound(err) {
		return Practice{Examples: []PracticeExample{}, Blocks: []PracticeBlock{}}, nil
	}

	if err != nil {
		return Practice{}, err
	}

	return practiceOf(row)
}

// Leave is the learner going away from the page of a rule: the sitting is kept
// for practiceGrace and then goes. Nothing to end is not an error.
func (h PracticeHandler) Leave(ctx context.Context, userID uuid.UUID, ruleID string) error {
	return h.q.LeaveLearningRulePractice(ctx, db.LeaveLearningRulePracticeParams{
		UserID: ids.ToPg(userID), RuleID: ruleID, ExpiresAt: db.Time(h.clock.Now().Add(practiceGrace)),
	})
}

// ensure makes the sitting with the first thing written into it.
func (h PracticeHandler) ensure(ctx context.Context, q *db.Queries, userID uuid.UUID, rule RuleSubject, now time.Time) (db.LearningRulePractice, error) {
	return q.EnsureLearningRulePractice(ctx, db.EnsureLearningRulePracticeParams{
		ID:                  ids.ToPg(ids.New()),
		UserID:              ids.ToPg(userID),
		RuleID:              rule.ID,
		Language:            rule.Language,
		TranslationLanguage: rule.Explanation,
		Level:               rule.Level,
		Title:               rule.Title,
		Theory:              rule.Theory,
		CreatedAt:           db.Time(now),
		ExpiresAt:           db.Time(now.Add(practiceIdle)),
	})
}

// current is the sitting the learner has on a rule, if it is still alive.
func (h PracticeHandler) current(ctx context.Context, userID uuid.UUID, ruleID string, now time.Time) (db.LearningRulePractice, bool, error) {
	if err := h.sweep(ctx, userID, now); err != nil {
		return db.LearningRulePractice{}, false, err
	}

	row, err := h.q.GetLearningRulePractice(ctx, db.GetLearningRulePracticeParams{UserID: ids.ToPg(userID), RuleID: ruleID})
	if db.IsNotFound(err) {
		return db.LearningRulePractice{}, false, nil
	}

	return row, err == nil, err
}

// AddExamples has the learner's agent write a few more sentences for the rule
// and adds them to the sitting. What was shown is never written twice.
func (h PracticeHandler) AddExamples(ctx context.Context, userID uuid.UUID, rule RuleSubject) (Practice, error) {
	now := h.clock.Now()

	row, found, err := h.current(ctx, userID, rule.ID, now)
	if err != nil {
		return Practice{}, err
	}

	var shown []PracticeExample

	if found {
		if shown, err = decodeExamples(row.Examples); err != nil {
			return Practice{}, err
		}
	}

	if len(shown) >= maxExamples {
		return Practice{}, apperr.NewConflict("that is enough sentences for this rule: practise them or leave the page for new ones")
	}

	seen := make([]string, len(shown))
	for i, e := range shown {
		seen[i] = e.Text
	}

	var (
		set   exampleSetOut
		fresh []exampleOut
	)

	if err := h.agent.completeJSON(ctx, userID, llm.PurposePracticeExamples, examplesSystem, examplesPrompt(rule, seen), &set, func() error {
		var err error
		fresh, err = cleanExamples(set.Examples, seen)

		return err
	}); err != nil {
		return Practice{}, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return Practice{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)

	row, err = h.ensure(ctx, tq, userID, rule, now)
	if err != nil {
		return Practice{}, err
	}

	// Under the lock, so two presses of the button cannot write over each other.
	row, err = tq.LockLearningRulePractice(ctx, db.LockLearningRulePracticeParams{ID: row.ID, UserID: ids.ToPg(userID)})
	if err != nil {
		return Practice{}, err
	}

	list, err := decodeExamples(row.Examples)
	if err != nil {
		return Practice{}, err
	}

	have := map[string]bool{}
	for _, e := range list {
		have[sentenceKey(e.Text)] = true
	}

	for _, e := range fresh {
		if len(list) >= maxExamples {
			break
		}

		if have[sentenceKey(e.Text)] {
			continue
		}

		have[sentenceKey(e.Text)] = true
		list = append(list, PracticeExample{ID: "e" + strconv.Itoa(len(list)+1), Text: e.Text, Translation: e.Translation})
	}

	raw, err := json.Marshal(list)
	if err != nil {
		return Practice{}, err
	}

	row, err = tq.SaveLearningRulePracticeExamples(ctx, db.SaveLearningRulePracticeExamplesParams{
		ID: row.ID, UserID: row.UserID, Examples: raw, UpdatedAt: db.Time(now), ExpiresAt: db.Time(now.Add(practiceIdle)),
	})
	if err != nil {
		return Practice{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Practice{}, err
	}

	return practiceOf(row)
}

func decodeExamples(raw []byte) ([]PracticeExample, error) {
	list := []PracticeExample{}

	return list, json.Unmarshal(raw, &list)
}

// WriteExercises has the learner's agent write a fresh set of drills for the
// rule. They replace the ones the sitting had.
func (h PracticeHandler) WriteExercises(ctx context.Context, userID uuid.UUID, rule RuleSubject) (Practice, error) {
	now := h.clock.Now()

	var set blockSetOut

	if err := h.agent.completeJSON(ctx, userID, llm.PurposePracticeDrills, practiceSystem, rulePrompt(rule, nil), &set, func() error {
		return validateBlocks(set.Blocks)
	}); err != nil {
		return Practice{}, err
	}

	blocks := make([]PracticeBlock, len(set.Blocks))

	for i, b := range set.Blocks {
		blocks[i] = PracticeBlock{ID: "b" + strconv.Itoa(i+1), Kind: b.Kind, Instruction: b.Instruction, Items: make([]PracticeItem, len(b.Items))}

		for j, it := range b.Items {
			options := it.Options
			if options == nil {
				options = []string{}
			}

			blocks[i].Items[j] = PracticeItem{
				ID: "b" + strconv.Itoa(i+1) + "i" + strconv.Itoa(j+1), Text: it.Text, Hint: it.Hint,
				Options: options, Answers: it.Answers, Note: it.Note,
			}
		}
	}

	raw, err := json.Marshal(blocks)
	if err != nil {
		return Practice{}, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return Practice{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tq := h.q.WithTx(tx)

	row, err := h.ensure(ctx, tq, userID, rule, now)
	if err != nil {
		return Practice{}, err
	}

	row, err = tq.SaveLearningRulePracticeExercises(ctx, db.SaveLearningRulePracticeExercisesParams{
		ID: row.ID, UserID: row.UserID, Exercises: raw, UpdatedAt: db.Time(now), ExpiresAt: db.Time(now.Add(practiceIdle)),
	})
	if err != nil {
		return Practice{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Practice{}, err
	}

	return practiceOf(row)
}

// item finds one line of the sitting's drills.
func (h PracticeHandler) item(ctx context.Context, userID uuid.UUID, ruleID, blockID, itemID string) (PracticeItem, error) {
	row, found, err := h.current(ctx, userID, ruleID, h.clock.Now())
	if err != nil {
		return PracticeItem{}, err
	}

	if !found {
		return PracticeItem{}, apperr.NewNotFound("these exercises are gone: write new ones")
	}

	p, err := practiceOf(row)
	if err != nil {
		return PracticeItem{}, err
	}

	for _, b := range p.Blocks {
		if b.ID != blockID {
			continue
		}

		for _, it := range b.Items {
			if it.ID == itemID {
				return it, nil
			}
		}
	}

	return PracticeItem{}, apperr.NewNotFound("exercise not found")
}

// Check grades one line by its key. Nothing is asked of an agent: the key was
// written with the exercise.
func (h PracticeHandler) Check(ctx context.Context, userID uuid.UUID, ruleID, blockID, itemID, answer string) (Check, error) {
	answer = strings.TrimSpace(answer)
	if answer == "" || utf8.RuneCountInString(answer) > maxAnswerRunes {
		return Check{}, apperr.NewInvalid("the answer must be 1 to 300 characters")
	}

	it, err := h.item(ctx, userID, ruleID, blockID, itemID)
	if err != nil {
		return Check{}, err
	}

	given := normAnswer(answer)
	out := Check{Key: it.Answers[0], Note: it.Note}

	for _, a := range it.Answers {
		if normAnswer(a) == given {
			out.Correct = true

			break
		}
	}

	return out, nil
}

// Key is the key of one line, for a learner who gives up on it.
func (h PracticeHandler) Key(ctx context.Context, userID uuid.UUID, ruleID, blockID, itemID string) (Check, error) {
	it, err := h.item(ctx, userID, ruleID, blockID, itemID)
	if err != nil {
		return Check{}, err
	}

	return Check{Key: it.Answers[0], Note: it.Note}, nil
}
