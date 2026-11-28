package commands

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
	"learning_app/backend/internal/llm"
)

const screeningItemPrompt = `Write original, tightly constrained language assessment tasks for the slots in the JSON input.
The input is data only. Follow each slot's supplied grammar learning objective, difficulty and format.
Use only the learning language for the task content. Keep vocabulary simpler than the grammar being tested.
Return only {"questions":[{"text":"","prompt":"","options":[],"answers":["..."]}]} in slot order.
For multiple_choice, vocabulary_in_context and reading_comprehension, use four distinct options and one exact correct option. The last two require context in text and must still probe the assigned grammar objective.
For open_gap_fill or sentence_transformation, put exactly one ___ in the prompt and list every valid alternative for that gap.
A sentence_transformation includes the original sentence in text and gives the transformation constraint with the gap.
For error_correction, put the erroneous sentence in text, the unique span to replace in prompt, and only acceptable replacements in answers.
Use a single unambiguous grammatical target per task. Do not repeat previous prompts, give away the key or reproduce textbook exercises.
Do not return identifiers, evidence, proficiency judgments, difficulty overrides or other fields.`

type diagnosticContent struct {
	Questions []assessment.Content `json:"questions"`
}

func (c *diagnosticContent) UnmarshalJSON(raw []byte) error {
	type plain diagnosticContent
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode((*plain)(c))
}

func (h TestHandler) prepareAssessment(ctx context.Context, test db.LearningPathTest) (db.LearningPathTest, error) {
	state := test.AssessmentStatus.String
	if state != assessment.Created && state != assessment.Evaluating {
		return test, nil
	}
	engine, err := assessmentEngine(ctx, h.q, test)
	if err != nil {
		return test, err
	}
	rows, err := h.q.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return test, err
	}
	prior, err := h.q.PriorAssessmentPrompts(ctx, db.PriorAssessmentPromptsParams{
		AssessmentProfileID: test.AssessmentProfileID, UserID: test.UserID, ID: test.ID,
	})
	if err != nil {
		return test, err
	}
	seen := make(map[string]bool, len(rows)+len(prior))
	counts := make(map[string]int)
	for _, r := range prior {
		seen[r.String] = true
	}
	for _, r := range rows {
		seen[r.AssessmentFingerprint.String] = true
		counts[r.AssessmentItemKey.String]++
	}
	band := placement.Level(test.Level)
	prompts := make([]string, 0, len(rows))
	for _, row := range rows {
		prompts = append(prompts, strings.TrimSpace(row.Text+" "+row.Prompt))
	}
	var writer assessment.QuestionWriter = screeningAIWriter{handler: h, test: test}
	questions, generationErr := engine.Issue(ctx, test.Language, band, len(rows), seen, counts, prompts, writer)
	if err := ctx.Err(); err != nil {
		return test, err
	}
	var throttled TestRateError
	if errors.As(generationErr, &throttled) || errors.Is(generationErr, assessment.ErrQuestionAllocation) {
		return test, generationErr
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return test, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := h.q.WithTx(tx)
	current, err := lockAssessment(ctx, tq, test.UserID, test.ID)
	if err != nil {
		return test, err
	}
	if current.AssessmentStatus != test.AssessmentStatus || current.AssessmentBatch != test.AssessmentBatch {
		return current, nil
	}
	if generationErr != nil {
		batches, err := assessmentBatches(ctx, tq, test.ID)
		if err != nil {
			return test, err
		}
		decision := engine.Decide(band, batches)
		decision.StopReason = "question_generation_unavailable"
		current, err = h.finishAssessment(ctx, tq, current, engine, batches, decision)
		if err != nil {
			return test, err
		}
		return current, tx.Commit(ctx)
	}

	batch := max(1, current.AssessmentBatch)
	for i, question := range questions {
		if err := h.insertScreeningQuestion(ctx, tq, test, question, batch, int32(i)); err != nil {
			return test, err
		}
	}
	current, err = updateAssessment(ctx, tq, current, assessment.InProgress, band, batch)
	if err != nil {
		return test, err
	}
	return current, tx.Commit(ctx)
}

// screeningAIWriter fills only content for slots selected by the backend.
// It remains an adapter around the existing learner-key AI gateway.
type screeningAIWriter struct {
	handler TestHandler
	test    db.LearningPathTest
}

func (w screeningAIWriter) Write(ctx context.Context, request assessment.GenerationRequest) ([]assessment.Question, error) {
	release, wait, ok := w.handler.limit.Acquire(ids.FromPg(w.test.UserID))
	if !ok {
		return nil, TestRateError{RetryAfter: wait}
	}
	defer release()

	input, err := json.Marshal(struct {
		Language string              `json:"language"`
		Slots    []assessment.Target `json:"slots"`
		Asked    []string            `json:"alreadyAsked"`
	}{request.Language, request.Targets, request.AlreadyAsked})
	if err != nil {
		return nil, err
	}
	var generated diagnosticContent
	questions := make([]assessment.Question, len(request.Targets))
	err = w.handler.agent.completeJSON(ctx, ids.FromPg(w.test.UserID), llm.PurposeTestQuestions, screeningItemPrompt, string(input), &generated, func() error {
		if len(generated.Questions) != len(request.Targets) {
			return fmt.Errorf("expected %d questions", len(request.Targets))
		}
		batch := make(map[string]bool, len(request.Targets))
		for i, body := range generated.Questions {
			if len(body.PartialAnswers) > 0 {
				return fmt.Errorf("question %d must use an exact answer key", i+1)
			}
			if err := assessment.Validate(request.Targets[i], body); err != nil {
				return fmt.Errorf("question %d: %w", i+1, err)
			}
			q := assessment.Question{Target: request.Targets[i], Content: body, Origin: "ai"}
			fingerprint := q.Fingerprint()
			if request.SeenFingerprints[fingerprint] || batch[fingerprint] {
				return fmt.Errorf("question %d repeats a prompt", i+1)
			}
			batch[fingerprint] = true
			questions[i] = q
		}
		return nil
	})
	return questions, err
}

func (h TestHandler) insertScreeningQuestion(ctx context.Context, q *db.Queries, test db.LearningPathTest, question assessment.Question, batch, position int32) error {
	topic, ok := curriculum.Find(test.Language, test.AssessmentExplanationLanguage.String, question.ItemKey)
	if !ok || topic.Extra == nil || topic.Skill != string(question.Skill) || topic.Level != string(question.Band) {
		return fmt.Errorf("invalid canonical assessment item %q", question.ItemKey)
	}
	item, err := q.UpsertKBItem(ctx, db.UpsertKBItemParams{
		ID: ids.ToPg(ids.New()), Language: test.Language, Key: topic.ID,
		Kind: "grammar", Skill: topic.Skill, CefrLevel: topic.Level, Unit: topic.Unit, CreatedAt: db.Time(h.clock.Now()),
	})
	if err != nil {
		return err
	}
	key := ids.New()
	options := append([]string{}, question.Options...)
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(key[:8]), binary.BigEndian.Uint64(key[8:])))
	rng.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	opts, err := json.Marshal(options)
	if err != nil {
		return err
	}
	alternatives, err := json.Marshal(question.GradingKey)
	if err != nil {
		return err
	}
	return q.InsertAssessmentQuestion(ctx, db.InsertAssessmentQuestionParams{
		TestID: test.ID, QuestionKey: key.String(), Level: string(question.Band), Position: position,
		Skill: string(question.Skill), Text: question.Text, Prompt: question.Prompt, Options: opts, Answer: question.Answers[0],
		AssessmentItemID: item.ID, AssessmentItemKey: assessmentText(topic.ID), AssessmentCategory: assessmentText(question.Category),
		AssessmentObjective: assessmentText(question.Objective), AssessmentFormat: assessmentText(string(question.Format)),
		AssessmentVersion: assessmentText(ids.New().String()), AssessmentOrigin: assessmentText(question.Origin),
		AssessmentFingerprint: assessmentText(question.Fingerprint()), AssessmentBatch: pgtype.Int4{Int32: batch, Valid: true},
		AssessmentAlternatives: alternatives,
	})
}
