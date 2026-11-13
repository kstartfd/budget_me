package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEveryExplanationPromptAsksForMarkedExamples: the page sets backticked
// text in a font of its own, so each prompt that makes text for the learner to
// read has to ask for the marks.
func TestEveryExplanationPromptAsksForMarkedExamples(t *testing.T) {
	for name, system := range map[string]string{
		"analyze": analyzeSystem, "plan": planSystem, "exercises": exercisesSystem,
		"review": reviewSystem, "practice": practiceSystem,
	} {
		if !strings.Contains(system, "wrapping it in backticks") {
			t.Errorf("%s does not ask for examples in backticks", name)
		}

		if !strings.Contains(system, "Never use backticks in text, prompt, options, answer") {
			t.Errorf("%s does not keep the marks out of the tasks", name)
		}
	}

	if !strings.Contains(reviewSystem, "quote the learner's wording in backticks") {
		t.Error("the review does not ask for the quoted wording in backticks")
	}
}

// TestTheTaskOfAnExerciseNeverCarriesTheMark: a backtick the writer slips into
// what the learner works on would show in the task and spoil the comparison
// with the answer; only the rule, which is read, keeps its marked examples.
func TestTheTaskOfAnExerciseNeverCarriesTheMark(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"exercises": []map[string]any{{
		"kind": "choice", "text": "He `goes` home.", "prompt": "Pick the `right` form.",
		"options": []string{"`goes`", "go", "going", "gone"}, "answer": "`goes`",
		"rule": "After he we add -s: `He goes home.`",
	}}})

	got, err := parseExercises(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	e := got[0]
	if e.Text != "He goes home." || e.Prompt != "Pick the right form." || e.Answer != "goes" || e.Options[0] != "goes" {
		t.Errorf("the task kept the mark: %+v", e)
	}

	if e.Rule != "After he we add -s: `He goes home.`" {
		t.Errorf("the rule lost its marked example: %q", e.Rule)
	}
}

func TestTheItemsOfADrillAndItsExamplesLoseTheMarkButTheNoteKeepsIt(t *testing.T) {
	it := itemOut{Text: "She ... (`go`) home.", Hint: "`go`", Answers: []string{"`goes`"}, Note: "After she: `She goes.`"}
	if err := validateItem(blockForm, &it); err != nil {
		t.Fatal(err)
	}

	if it.Text != "She ... (go) home." || it.Hint != "go" || it.Answers[0] != "goes" {
		t.Errorf("the item kept the mark: %+v", it)
	}

	pick := itemOut{Text: "She ... home.", Options: []string{"`goes`", "go"}, Answers: []string{"`goes`"}}
	if err := validateItem(blockChoose, &pick); err != nil {
		t.Fatal(err)
	}

	if pick.Options[0] != "goes" || pick.Answers[0] != "goes" {
		t.Errorf("the options kept the mark: %+v", pick)
	}

	if it.Note != "After she: `She goes.`" {
		t.Errorf("the note lost its marked example: %q", it.Note)
	}

	list, err := cleanExamples([]exampleOut{{Text: "She `goes` home.", Translation: "Она идёт домой."}}, nil)
	if err != nil || list[0].Text != "She goes home." {
		t.Errorf("the example kept the mark: %+v %v", list, err)
	}
}
