package assessment

import (
	"context"

	"learning_app/backend/internal/learningpath/placement"
)

// QuestionSource allocates target slots. It cannot set the assessment result;
// generated wording is validated against each chosen target before storage.
type QuestionSource interface {
	Select(band placement.Level, count int, seen map[string]bool, itemCounts map[string]int) []Slot
}

// BankQuestionSource uses canonical, reviewed grammar targets for Quick. A slot
// with no unused bank wording can be filled by the AI content writer.
type BankQuestionSource struct{ Questions []Question }

func (s BankQuestionSource) Select(band placement.Level, count int, seen map[string]bool, itemCounts map[string]int) []Slot {
	return Select(s.Questions, band, count, seen, itemCounts)
}

// GenerationRequest contains only backend-chosen targets and exposure context.
// A writer may supply wording and keys, but cannot choose what is measured.
type GenerationRequest struct {
	Language         string
	Targets          []Target
	AlreadyAsked     []string
	SeenFingerprints map[string]bool
}

type QuestionWriter interface {
	Write(context.Context, GenerationRequest) ([]Question, error)
}

// AnswerEvaluator grades an issued question without deciding proficiency or
// moving its difficulty. Skills with rubric/audio policies can supply a
// different evaluator only when their complete workflow is implemented.
type AnswerEvaluator interface {
	Evaluate(question Question, answer string, skip bool) (Outcome, error)
}

type KeyedAnswerEvaluator struct{}

func (KeyedAnswerEvaluator) Evaluate(question Question, answer string, skip bool) (Outcome, error) {
	return Grade(question, answer, skip)
}
