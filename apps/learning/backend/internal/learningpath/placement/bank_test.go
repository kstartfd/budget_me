package placement_test

import (
	"testing"

	"learning_app/backend/internal/learningpath/placement"
)

// TestBankIsWellFormed guards the data every level test draws from: a broken
// question would grade a learner wrong, so the bank must be sound by
// construction.
func TestBankIsWellFormed(t *testing.T) {
	if !placement.Has("en") {
		t.Fatal("the English ladder is gone")
	}

	seen := map[string]bool{}

	for _, level := range placement.Levels {
		questions := placement.Questions("en", level)
		if len(questions) < placement.BatchSize {
			t.Fatalf("%s has %d questions, a batch needs %d", level, len(questions), placement.BatchSize)
		}

		for _, q := range questions {
			if seen[q.Key] {
				t.Fatalf("duplicate key %q", q.Key)
			}
			seen[q.Key] = true

			if q.Prompt == "" {
				t.Fatalf("%s has an empty prompt", q.Key)
			}

			if len(q.Options) != 4 {
				t.Fatalf("%s has %d options, want 4", q.Key, len(q.Options))
			}

			found := false
			for _, o := range q.Options {
				if o == "" {
					t.Fatalf("%s has an empty option", q.Key)
				}
				if placement.Grade(q, o) && o != q.Answer {
					t.Fatalf("%s grades a wrong option right: %q", q.Key, o)
				}
				if o == q.Answer {
					found = true
				}
			}

			if !found {
				t.Fatalf("%s: the answer is not among the options", q.Key)
			}

			if !placement.Grade(q, q.Answer) {
				t.Fatalf("%s: the answer does not grade itself right", q.Key)
			}

			switch q.Skill {
			case placement.SkillReading:
			case placement.SkillGrammar, placement.SkillVocabulary:
				if q.Text != "" {
					t.Fatalf("%s is %s but carries a reading passage", q.Key, q.Skill)
				}
			default:
				t.Fatalf("%s has unknown skill %q", q.Key, q.Skill)
			}
		}
	}

	// A batch must be drawable for every level and unique within a test.
	for _, level := range placement.Levels {
		drawn := placement.Batch("en", level, nil, placement.BatchSize)
		if len(drawn) != placement.BatchSize {
			t.Fatalf("%s drew %d questions, want %d", level, len(drawn), placement.BatchSize)
		}

		exclude := map[string]bool{}
		for _, q := range drawn {
			exclude[q.Key] = true
		}

		rest := placement.Batch("en", level, exclude, placement.BatchSize)
		for _, q := range rest {
			if exclude[q.Key] {
				t.Fatalf("%s was drawn twice for one test", q.Key)
			}
		}
	}
}

func TestLadderEdges(t *testing.T) {
	if next, ok := placement.Next(placement.A1); !ok || next != placement.A2 {
		t.Fatalf("above A1: %q %t", next, ok)
	}

	if _, ok := placement.Next(placement.C2); ok {
		t.Fatal("the ladder must end at C2")
	}

	if below := placement.Below(placement.A1); below != placement.A1 {
		t.Fatalf("below A1 must clamp at A1, got %q", below)
	}

	if below := placement.Below(placement.C2); below != placement.C1 {
		t.Fatalf("below C2: %q", below)
	}
}
