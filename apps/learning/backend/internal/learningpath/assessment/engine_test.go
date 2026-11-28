package assessment

import (
	"context"
	"errors"
	"testing"

	"learning_app/backend/internal/learningpath/placement"
)

type testWriter func(context.Context, GenerationRequest) ([]Question, error)

func (w testWriter) Write(ctx context.Context, request GenerationRequest) ([]Question, error) {
	return w(ctx, request)
}

func TestSectionEngineGrammarJourney(t *testing.T) {
	policy := QuickPolicy()
	mode, ok := RuntimeMode(QuickScreening, policy)
	if !ok {
		t.Fatal("grammar mode unavailable")
	}
	bank, ok := Bank("en", "en")
	if !ok {
		t.Fatal("grammar bank unavailable")
	}
	engine, err := NewSectionEngine(policy, mode, bank)
	if err != nil {
		t.Fatal(err)
	}
	questions, err := engine.Issue(context.Background(), "en", placement.A2, 0, map[string]bool{}, map[string]int{}, nil, nil)
	if err != nil || len(questions) != policy.BatchSize {
		t.Fatalf("issue: %d %v", len(questions), err)
	}
	if got, err := engine.Evaluate(questions[0], "", true); err != nil || got != Skipped {
		t.Fatalf("skip: %s %v", got, err)
	}
	if got, err := engine.Evaluate(questions[0], questions[0].Answers[0], false); err != nil || got != Correct {
		t.Fatalf("answer: %s %v", got, err)
	}
	facts := []Observation{{Target: questions[0].Target, Outcome: Skipped, Origin: "curated"}}
	batches := []Batch{{Band: placement.A2, Issued: len(questions), Observations: facts}}
	decision := engine.Decide(placement.A2, batches)
	decision.StopReason = "learner_stopped"
	result, status, err := engine.Result("en", batches, decision)
	if err != nil || status != Partial || len(result.Sections) != 1 || result.Sections[0].Skill != "grammar" || result.DifficultyRange != nil {
		t.Fatalf("partial result: %s %+v %v", status, result.Sections, err)
	}
	if _, err := engine.Issue(context.Background(), "en", placement.A2, policy.MaxQuestions, map[string]bool{}, map[string]int{}, nil, nil); !errors.Is(err, ErrQuestionAllocation) {
		t.Fatalf("accepted allocation beyond policy: %v", err)
	}
}

func TestSectionEngineRejectsWriterChangingTheTarget(t *testing.T) {
	policy := QuickPolicy()
	mode, _ := RuntimeMode(QuickScreening, policy)
	bank, _ := Bank("en", "en")
	engine, err := NewSectionEngine(policy, mode, bank)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, item := range bank {
		seen[item.Fingerprint()] = true
	}
	writer := testWriter(func(_ context.Context, req GenerationRequest) ([]Question, error) {
		result := make([]Question, len(req.Targets))
		for i, target := range req.Targets {
			result[i] = Question{Target: target, Content: Content{Prompt: "Different task", Options: []string{"a", "b", "c", "d"}, GradingKey: GradingKey{Answers: []string{"a"}}}}
			result[i].ItemKey = "unapproved-item"
		}
		return result, nil
	})
	if _, err := engine.Issue(context.Background(), "en", placement.A2, 0, seen, map[string]int{}, nil, writer); err == nil {
		t.Fatal("writer changed the backend-selected target")
	}
}
