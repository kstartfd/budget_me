package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestListeningIsAskedInTheLearnedLanguageFromA2: a beginner picks among
// answers in the language of the explanations, from A2 the question and the
// answers are in the language being learned, whatever the explanations are in.
func TestListeningIsAskedInTheLearnedLanguageFromA2(t *testing.T) {
	for _, tc := range []struct {
		level, explain, want string
	}{
		{"A1", "Russian", "Russian"},
		{"A2", "Russian", "English"},
		{"B1", "Russian", "English"},
		{"B2", "Russian", "English"},
		{"C1", "Russian", "English"},
		{"A1", "English", "English"},
		{"", "Russian", "Russian"},
	} {
		_, prompt := exercisesPrompt("English", tc.explain, "Russian", tc.level, "listening", "Numbers", "Hear numbers", "", nil, 0, 0)

		var data map[string]any
		if err := json.Unmarshal([]byte(prompt), &data); err != nil {
			t.Fatal(err)
		}

		if data["listenLanguage"] != tc.want {
			t.Errorf("level %q explained in %s: listening in %v, want %s", tc.level, tc.explain, data["listenLanguage"], tc.want)
		}

		if data["explanationLanguage"] != tc.explain || data["nativeLanguage"] != "Russian" {
			t.Errorf("the languages of the explanation and of the learner must travel with the topic: %v", data)
		}
	}
}

// TestTheWriterIsToldWhichLanguageEachPartIsIn: the instruction follows the
// learner's choice of explanation language, the rest of an exercise is in the
// learned language, and a listening exercise has a language of its own.
func TestTheWriterIsToldWhichLanguageEachPartIsIn(t *testing.T) {
	for _, want := range []string{
		"opens with a short task instruction in explanationLanguage and never in the other language",
		"The sentence to translate is in the learner's own language (nativeLanguage)",
		"its question in prompt and all four of its options are in listenLanguage",
	} {
		if !strings.Contains(exercisesSystem, want) {
			t.Errorf("the exercise writer is not told %q", want)
		}
	}

	// The two old statements that every prompt is in the learned language
	// contradicted the instruction in the explanation language.
	if strings.Contains(exercisesSystem, "text, prompt and options — in the learned language") {
		t.Error("the writer is still told to write the whole prompt in the learned language")
	}
}
