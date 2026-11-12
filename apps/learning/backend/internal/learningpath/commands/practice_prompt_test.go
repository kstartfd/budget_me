package commands

import (
	"strings"
	"testing"
)

func TestNormAnswerForgivesWhatIsNotGraded(t *testing.T) {
	same := [][2]string{
		{"Had she left?", "had she left"},
		{"  He HAD gone  home. ", "he had gone home"},
		{"Don’t go", "don't go"},
		{"Yes, I did!", "yes i did"},
	}

	for _, pair := range same {
		if normAnswer(pair[0]) != normAnswer(pair[1]) {
			t.Errorf("%q and %q must grade the same: %q vs %q", pair[0], pair[1], normAnswer(pair[0]), normAnswer(pair[1]))
		}
	}

	// What does change the answer is kept: a contraction is not the full form.
	if normAnswer("don't") == normAnswer("do not") {
		t.Error("a contraction and its full form are told apart, so both must be listed as answers")
	}

	if normAnswer("had gone") == normAnswer("have gone") {
		t.Error("a different word is a different answer")
	}
}

func formItem() itemOut {
	return itemOut{Text: "They had ... the film by six.", Hint: "see", Answers: []string{"seen"}, Note: "had + V3"}
}

func block(kind string, items ...itemOut) blockOut {
	return blockOut{Kind: kind, Instruction: "Do it.", Items: items}
}

func three(it itemOut) []itemOut { return []itemOut{it, it, it} }

func TestValidateBlocksAcceptsTheFourKinds(t *testing.T) {
	blocks := []blockOut{
		block(blockForm, three(formItem())...),
		block(blockChoose, three(itemOut{Text: "She had ... lunch.", Options: []string{"eat", "eaten"}, Answers: []string{"eaten"}})...),
		block(blockCorrect, three(itemOut{Text: "He had went.", Answers: []string{"He had gone."}})...),
		block(blockMake, three(itemOut{Text: "She had left.", Hint: "question", Answers: []string{"Had she left?"}})...),
	}

	if err := validateBlocks(blocks); err != nil {
		t.Fatalf("a good set was refused: %v", err)
	}
}

func TestValidateBlocksRefusesWhatCannotBeGraded(t *testing.T) {
	cases := map[string]blockOut{
		"unknown kind":       {Kind: "essay", Instruction: "x", Items: three(formItem())},
		"no instruction":     {Kind: blockForm, Items: three(formItem())},
		"too few items":      block(blockForm, formItem()),
		"no gap in a form":   block(blockForm, three(itemOut{Text: "They had seen it.", Hint: "see", Answers: []string{"seen"}})...),
		"two gaps":           block(blockForm, three(itemOut{Text: "They ... the ... film.", Hint: "see", Answers: []string{"seen"}})...),
		"form without hint":  block(blockForm, three(itemOut{Text: "They had ... it.", Answers: []string{"seen"}})...),
		"make without hint":  block(blockMake, three(itemOut{Text: "She left.", Answers: []string{"Did she leave?"}})...),
		"no answer":          block(blockForm, three(itemOut{Text: "They had ... it.", Hint: "see"})...),
		"choose, one option": block(blockChoose, three(itemOut{Text: "She had ... it.", Options: []string{"eaten"}, Answers: []string{"eaten"}})...),
		"choose, key absent": block(blockChoose, three(itemOut{Text: "She had ... it.", Options: []string{"eat", "ate"}, Answers: []string{"eaten"}})...),
		"choose, repeats":    block(blockChoose, three(itemOut{Text: "She had ... it.", Options: []string{"eaten", "Eaten"}, Answers: []string{"eaten"}})...),
		"options on a make":  block(blockMake, three(itemOut{Text: "She left.", Hint: "question", Options: []string{"a", "b"}, Answers: []string{"Did she leave?"}})...),
		"overlong text":      block(blockForm, three(itemOut{Text: strings.Repeat("a ", 200) + "...", Hint: "see", Answers: []string{"seen"}})...),
	}

	for name, b := range cases {
		if err := validateBlocks([]blockOut{b}); err == nil {
			t.Errorf("%s: a set that cannot be graded was accepted", name)
		}
	}

	if err := validateBlocks(nil); err == nil {
		t.Error("an empty set was accepted")
	}
}

func TestValidateBlocksKeepsOnlyTheKeyOfAChoice(t *testing.T) {
	blocks := []blockOut{block(blockChoose, three(itemOut{Text: "She had ... it.", Options: []string{"eat", "eaten"}, Answers: []string{"eaten", "ate"}})...)}

	if err := validateBlocks(blocks); err != nil {
		t.Fatal(err)
	}

	// "ate" is not among the options but would have been accepted as right.
	if got := blocks[0].Items[0].Answers; len(got) != 1 || got[0] != "eaten" {
		t.Fatalf("a choice has one right option: %v", got)
	}
}

func TestCleanExamplesDropsWhatWasSeen(t *testing.T) {
	seen := []string{"She had already left."}

	got, err := cleanExamples([]exampleOut{
		{Text: "She had already left!", Translation: "Она уже ушла."}, // seen, punctuation aside
		{Text: "They had eaten.", Translation: "Они поели."},
		{Text: "they had eaten", Translation: "Они поели."}, // repeats within the batch
		{Text: "He had won.", Translation: "Он выиграл."},
	}, seen)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].Text != "They had eaten." || got[1].Text != "He had won." {
		t.Fatalf("only new sentences stay: %+v", got)
	}

	if _, err := cleanExamples([]exampleOut{{Text: "She had already left.", Translation: "x"}}, seen); err == nil {
		t.Error("a batch with nothing new must be refused, so the agent is asked again")
	}

	if _, err := cleanExamples([]exampleOut{{Text: "Fine.", Translation: ""}}, nil); err == nil {
		t.Error("a sentence with no translation was accepted")
	}
}

func TestCleanExamplesKeepsAtMostOneBatch(t *testing.T) {
	var list []exampleOut
	for i := 0; i < 12; i++ {
		list = append(list, exampleOut{Text: strings.Repeat("a", i+1), Translation: "t"})
	}

	got, err := cleanExamples(list, nil)
	if err != nil || len(got) != examplesBatch {
		t.Fatalf("an over-eager agent gets cut to a batch: %d %v", len(got), err)
	}
}
