package commands

import (
	"strings"
	"testing"
)

func TestOrderExercise(t *testing.T) {
	ok := exercise{Kind: "order", Prompt: "Order the words", Options: []string{"al", "Voy", "mañana", "cine"}, Answer: "Voy al cine mañana | Mañana voy al cine", Rule: "ir a"}
	if err := validExercise(ok); err != nil {
		t.Fatalf("a good order exercise: %v", err)
	}

	for name, bad := range map[string]exercise{
		"already in order":                {Kind: "order", Prompt: "p", Options: []string{"Voy", "al", "cine"}, Answer: "Voy al cine", Rule: "r"},
		"a word is missing":               {Kind: "order", Prompt: "p", Options: []string{"al", "Voy", "cine"}, Answer: "Voy al cine mañana", Rule: "r"},
		"a second order uses other words": {Kind: "order", Prompt: "p", Options: []string{"al", "Voy", "cine"}, Answer: "Voy al cine | Vamos al cine", Rule: "r"},
		"too few words":                   {Kind: "order", Prompt: "p", Options: []string{"Voy", "ya"}, Answer: "ya Voy", Rule: "r"},
		"a phrase as option":              {Kind: "order", Prompt: "p", Options: []string{"Voy al", "cine", "ya"}, Answer: "Voy al cine ya", Rule: "r"},
	} {
		if err := validExercise(bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestOrderNeverHandsOverTheSentence(t *testing.T) {
	base := exercise{Kind: "order", Prompt: "Put the words in order", Options: []string{"al", "Voy", "mañana", "cine"}, Answer: "Voy al cine mañana", Rule: "r"}

	withText := base
	withText.Text = "Voy al cine mañana."

	withPrompt := base
	withPrompt.Prompt = "Ordena: «Voy al cine, mañana»"

	for name, bad := range map[string]exercise{"in the clue": withText, "in the instruction": withPrompt} {
		if err := validExercise(bad); err == nil {
			t.Errorf("%s: the sentence was handed over", name)
		}
	}

	clue := base
	clue.Text = "Ich gehe morgen ins Kino."
	if err := validExercise(clue); err != nil {
		t.Errorf("a translation as the clue: %v", err)
	}
}

func TestOrderIsGradedByAnyOfTheOrdersTheKeyAllows(t *testing.T) {
	key := "Voy al cine mañana | Mañana voy al cine"
	for _, good := range []string{"Voy al cine mañana", "mañana voy al cine", "  Mañana  voy al cine "} {
		if !gradeOrder(key, good) {
			t.Errorf("%q must be right", good)
		}
	}

	for _, bad := range []string{"Voy mañana al cine", "Voy al cine"} {
		if gradeOrder(key, bad) {
			t.Errorf("%q must be wrong", bad)
		}
	}
}

func TestClozeExercise(t *testing.T) {
	text := "Ayer [1] a casa de mi abuela. Ella [2] una sopa muy rica y yo [3] mucho. Después los dos [4] la televisión hasta tarde, y por fin me fui a dormir."
	ok := exercise{Kind: "cloze", Prompt: "Fill the gaps", Text: text, Answer: "fui | hizo | comí | vimos", Rule: "indefinido"}

	if err := validExercise(ok); err != nil {
		t.Fatalf("a good cloze: %v", err)
	}

	for name, bad := range map[string]exercise{
		"too few answers":    {Kind: "cloze", Prompt: "p", Text: text, Answer: "fui | hizo | comí", Rule: "r"},
		"too short a text":   {Kind: "cloze", Prompt: "p", Text: "Ayer [1] y [2] y [3].", Answer: "a | b | c", Rule: "r"},
		"gaps out of order":  {Kind: "cloze", Prompt: "p", Text: strings.Replace(text, "[1]", "[2]", 1), Answer: "a | b | c | d", Rule: "r"},
		"options not wanted": {Kind: "cloze", Prompt: "p", Text: text, Options: []string{"a"}, Answer: "fui | hizo | comí | vimos", Rule: "r"},
		"an empty answer":    {Kind: "cloze", Prompt: "p", Text: text, Answer: "fui |  | comí | vimos", Rule: "r"},
	} {
		if err := validExercise(bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestGapsAreCountedOneByOne(t *testing.T) {
	key := "fui | hizo/cocinó | comí | vimos"

	for answer, wrong := range map[string]int{
		"fui | hizo | comí | vimos":         0,
		"Fui | cocinó | comí | vimos":       0,
		"fui | hizo | comi | vimos":         1,
		"fue | cocinaba | comí | vimos":     2,
		"fui | hizo":                        2,
		"":                                  4,
		"fui | hizo | comí | vimos | extra": 0,
	} {
		if got := gradeGaps(key, answer); got != wrong {
			t.Errorf("%q: %d wrong, want %d", answer, got, wrong)
		}
	}
}

func TestReadExercise(t *testing.T) {
	text := "Me llamo Ana y vivo en Sevilla con mi hermano. Trabajo en un banco desde hace tres años. Los sábados voy al mercado y compro fruta."
	ok := exercise{Kind: "read", Prompt: "Read and answer", Text: text, Options: []string{"¿Dónde vive Ana?", "¿Cuándo va al mercado?", "Ana trabaja en un ..."}, Answer: "En Sevilla | Los sábados | banco", Rule: "presente"}

	if err := validExercise(ok); err != nil {
		t.Fatalf("a good read exercise: %v", err)
	}

	ok.Answer = "En Sevilla | Los sábados"
	if err := validExercise(ok); err == nil {
		t.Error("a question without a model answer was accepted")
	}

	ok.Answer, ok.Options = "En Sevilla", []string{"¿Dónde vive Ana?"}
	if err := validExercise(ok); err == nil {
		t.Error("a single question was accepted")
	}
}

func TestDrillsAreWrittenAfterTheBookOfTheLanguage(t *testing.T) {
	for language, want := range map[string]string{"en": "English Grammar in Use", "en-US": "English Grammar in Use", "es": "Gramática de uso del español", "es-MX": "Gramática de uso del español"} {
		if !strings.Contains(practiceSystemFor(language), want) {
			t.Errorf("%s: the drills are not written after %q", language, want)
		}
	}

	if strings.Contains(practiceSystemFor("es"), "Murphy") {
		t.Error("the Spanish drills are written after an English book")
	}
}

func TestALeakedClueIsDroppedNotTheSet(t *testing.T) {
	list := []exercise{{Kind: "order", Prompt: "Put the words in order", Text: "Voy al cine mañana.", Options: []string{"al", "Voy", "mañana", "cine"}, Answer: "Voy al cine mañana", Rule: "r"}}
	if err := validateExercises(list); err != nil {
		t.Fatalf("the set was thrown away: %v", err)
	}

	if list[0].Text != "" {
		t.Errorf("the clue that gave the sentence away stayed: %q", list[0].Text)
	}
}
