package assessment

import (
	"context"
	"errors"
	"fmt"

	"learning_app/backend/internal/learningpath/placement"
)

var ErrQuestionAllocation = errors.New("invalid screening question allocation")

// SectionEngine coordinates the released grammar section. Storage and locking
// remain in the command layer; its inputs come from the persisted snapshot.
type SectionEngine struct {
	policy Policy
	mode   ModeBlueprint
	bank   []Question
}

func NewSectionEngine(policy Policy, mode ModeBlueprint, bank []Question) (SectionEngine, error) {
	if policy.Validate() != nil || mode.Validate() != nil || mode.Type != QuickScreening ||
		mode.Version != policy.Version || len(mode.Sections) != 1 ||
		mode.Sections[0].Skill != GrammarSkill || mode.Sections[0].Evaluator != KeyedEvaluator ||
		mode.Sections[0].MinQuestions != policy.MinQuestions || mode.Sections[0].MaxQuestions != policy.MaxQuestions {
		return SectionEngine{}, errors.New("invalid grammar section blueprint")
	}
	return SectionEngine{policy: policy, mode: mode, bank: bank}, nil
}

// Issue chooses targets first, then asks the writer only for missing wording.
// The writer cannot replace a target, choose a skill, or supply duplicate tasks.
func (e SectionEngine) Issue(ctx context.Context, language string, band placement.Level, issued int,
	seen map[string]bool, itemCounts map[string]int, asked []string, writer QuestionWriter) ([]Question, error) {
	count := min(e.policy.BatchSize, e.policy.MaxQuestions-issued)
	var source QuestionSource = BankQuestionSource{Questions: e.bank}
	slots := source.Select(band, count, seen, itemCounts)
	if count <= 0 || len(slots) != count {
		return nil, ErrQuestionAllocation
	}
	questions := make([]Question, len(slots))
	var targets []Target
	var missing []int
	for i, slot := range slots {
		if slot.Bank == nil {
			targets = append(targets, slot.Target)
			missing = append(missing, i)
			continue
		}
		questions[i] = *slot.Bank
		seen[slot.Bank.Fingerprint()] = true
	}
	if len(missing) == 0 {
		return questions, nil
	}
	if writer == nil {
		return nil, errors.New("assessment question writer unavailable")
	}
	generated, err := writer.Write(ctx, GenerationRequest{Language: language, Targets: targets, AlreadyAsked: asked, SeenFingerprints: seen})
	if err != nil {
		return nil, err
	}
	if len(generated) != len(targets) {
		return nil, fmt.Errorf("expected %d generated assessment questions", len(targets))
	}
	for i, question := range generated {
		if question.Target != targets[i] || Validate(question.Target, question.Content) != nil || seen[question.Fingerprint()] {
			return nil, fmt.Errorf("invalid generated assessment question %d", i+1)
		}
		seen[question.Fingerprint()] = true
		questions[missing[i]] = question
	}
	return questions, nil
}

func (e SectionEngine) Evaluate(question Question, answer string, skip bool) (Outcome, error) {
	var evaluator AnswerEvaluator = KeyedAnswerEvaluator{}
	return evaluator.Evaluate(question, answer, skip)
}

func (e SectionEngine) Decide(current placement.Level, batches []Batch) Decision {
	return Decide(e.policy, current, batches)
}

func (e SectionEngine) Result(language string, batches []Batch, decision Decision) (Result, string, error) {
	result := Summarize(e.policy, language, e.bank, batches, decision)
	sections := make([]SectionState, 0, len(result.Sections))
	for _, section := range result.Sections {
		sections = append(sections, SectionState{Skill: AssessedSkill(section.Skill), Status: section.Status})
	}
	state, err := AggregateMode(e.mode, sections)
	return result, state.Status, err
}
