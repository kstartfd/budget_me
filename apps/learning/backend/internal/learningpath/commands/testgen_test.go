package commands

import (
	"strings"
	"testing"

	"learning_app/backend/internal/learningpath/placement"
)

func goodQuestion(prompt string) generatedQuestion {
	return generatedQuestion{
		Skill:   "grammar",
		Prompt:  prompt,
		Options: []string{"had I closed", "I closed", "did I close", "I had closed"},
		Answer:  " Had I closed ",
	}
}

func TestValidateBatchAcceptsAndNormalises(t *testing.T) {
	batch := generatedBatch{Questions: []generatedQuestion{goodQuestion("No sooner ___ the door."), goodQuestion("Second ___.")}}

	if err := validateBatch(&batch, 2); err != nil {
		t.Fatal(err)
	}

	if batch.Questions[0].Answer != "had I closed" {
		t.Fatalf("the answer must be snapped to the option: %q", batch.Questions[0].Answer)
	}

	for _, q := range questionsOf(batch, placement.B2) {
		if !strings.HasPrefix(q.Key, "B2-") || !placement.Grade(q, q.Answer) {
			t.Fatalf("bad question: %+v", q)
		}
	}
}

func TestValidateBatchRefusesBrokenQuestions(t *testing.T) {
	cases := map[string]func(*generatedBatch){
		"wrong count":    func(b *generatedBatch) { b.Questions = b.Questions[:1] },
		"unknown skill":  func(b *generatedBatch) { b.Questions[0].Skill = "speaking" },
		"three options":  func(b *generatedBatch) { b.Questions[0].Options = b.Questions[0].Options[:3] },
		"answer missing": func(b *generatedBatch) { b.Questions[0].Answer = "nothing like it" },
		"duplicate option right": func(b *generatedBatch) {
			b.Questions[0].Options[1] = "had I closed"
		},
		"repeated prompt":      func(b *generatedBatch) { b.Questions[1].Prompt = b.Questions[0].Prompt },
		"reading without text": func(b *generatedBatch) { b.Questions[0].Skill = "reading" },
		"empty prompt":         func(b *generatedBatch) { b.Questions[0].Prompt = " " },
	}

	for name, break_ := range cases {
		batch := generatedBatch{Questions: []generatedQuestion{goodQuestion("One ___."), goodQuestion("Two ___.")}}
		break_(&batch)

		if err := validateBatch(&batch, 2); err == nil {
			t.Errorf("%s: a broken batch was accepted", name)
		}
	}
}

func TestValidateVerdict(t *testing.T) {
	cases := []struct {
		name           string
		v              verdict
		level          placement.Level
		asked, correct int
		ok             bool
	}{
		{"climb", verdict{"climb", "B1"}, placement.B1, 8, 6, true},
		{"climb on half", verdict{"Climb", "b1"}, placement.B1, 8, 4, true},
		{"climb on a mostly wrong batch", verdict{"climb", "B1"}, placement.B1, 8, 3, false},
		{"climb to another level", verdict{"climb", "B2"}, placement.B1, 8, 8, false},
		{"stop at the level", verdict{"stop", "B1"}, placement.B1, 8, 5, true},
		{"stop one under", verdict{"stop", "A2"}, placement.B1, 8, 2, true},
		{"stop two under", verdict{"stop", "A1"}, placement.B1, 8, 2, false},
		{"stop above", verdict{"stop", "B2"}, placement.B1, 8, 2, false},
		{"stop at A1 stays A1", verdict{"stop", "A1"}, placement.A1, 8, 0, true},
		{"stop on a perfect batch", verdict{"stop", "A2"}, placement.B1, 8, 8, false},
		{"not a level", verdict{"climb", "D9"}, placement.B1, 8, 8, false},
		{"unknown verdict", verdict{"maybe", "B1"}, placement.B1, 8, 8, false},
	}

	for _, c := range cases {
		v := c.v
		if err := validateVerdict(&v, c.level, c.asked, c.correct); (err == nil) != c.ok {
			t.Errorf("%s: err=%v, want ok=%t", c.name, err, c.ok)
		}
	}
}
