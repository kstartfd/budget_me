package learningpath_test

import (
	"net/http"
	"strings"
	"testing"
)

const levelsURL = "/api/v1/learning/path/levels"

func chooseLevel(t *testing.T, e env, language, level string) (int, map[string]any) {
	t.Helper()

	return request(t, e.e, "POST", levelsURL, e.session, map[string]any{"language": language, "translationLanguage": "ru", "level": level})
}

// TestAReachedLevelCanBePickedInsteadOfTheTest: after a reset the learner picks
// a level they already proved, and the program is built at once from it, with
// no test and no agent call to write one. A level they have not reached, a
// language they have no history in, a running program and nonsense are refused.
func TestAReachedLevelCanBePickedInsteadOfTheTest(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	// Nothing proved yet: not even A1 is free.
	if code, body := chooseLevel(t, e, "en", "A1"); code != http.StatusBadRequest {
		t.Fatalf("picking with no history: %d %v", code, body)
	}

	analyze(t, e, climb(t, e, "B1"))

	// A running program is not closed: finish or reset it first.
	if code, body := chooseLevel(t, e, "en", "A2"); code != http.StatusConflict {
		t.Fatalf("picking while a program runs: %d %v", code, body)
	}

	if code, body := request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"}); code != http.StatusOK {
		t.Fatalf("reset: %d %v", code, body)
	}

	for _, tc := range []struct{ language, level, what string }{
		{"en", "B2", "a level above the reached one"},
		{"en", "C2", "the top of the ladder"},
		{"en", "a2", "a level in the wrong case"},
		{"en", "", "no level"},
		{"pl", "A1", "a language with no history"},
	} {
		if code, body := chooseLevel(t, e, tc.language, tc.level); code != http.StatusBadRequest {
			t.Fatalf("picking %s: %d %v", tc.what, code, body)
		}
	}

	calls := len(e.provider.requests())

	code, body := chooseLevel(t, e, "en", "A2")
	if code != http.StatusCreated {
		t.Fatalf("picking A2: %d %v", code, body)
	}

	test := data(t, body)
	if test["status"] != "done" || test["levelReached"] != "A2" || test["kind"] != "placement" {
		t.Fatalf("the pick must be a finished placement at A2: %v", test)
	}

	if questions, _ := test["questions"].([]any); len(questions) != 0 {
		t.Fatalf("a pick has no questions: %v", test)
	}

	if len(e.provider.requests()) != calls {
		t.Fatal("picking a level must not spend the learner's agent")
	}

	path := analyze(t, e, test["id"].(string))
	if path["level"] != "A2" || path["source"] != "chosen" {
		t.Fatalf("the program must be built at the picked level and say so: %v", path)
	}

	var saidSo bool

	for _, prompt := range e.provider.requests() {
		saidSo = saidSo || strings.Contains(prompt, `questionsAnswered\":0`)
	}

	if !saidSo {
		t.Fatal("the planner must be told that no questions were answered")
	}

	// The reached level itself can be picked again too, and the history still
	// proves it: a second reset leaves both programs behind.
	if code, body := request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"}); code != http.StatusOK {
		t.Fatalf("second reset: %d %v", code, body)
	}

	if code, body := chooseLevel(t, e, "en", "B1"); code != http.StatusCreated || data(t, body)["levelReached"] != "B1" {
		t.Fatalf("picking the reached level: %d %v", code, body)
	}
}

// TestClearingTheHistoryTakesThePickableLevelsAway: the history is the proof of
// a level, so without it every level needs the test again.
func TestClearingTheHistoryTakesThePickableLevelsAway(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A2"))

	if code, body := request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"}); code != http.StatusOK {
		t.Fatalf("reset: %d %v", code, body)
	}

	if code, body := request(t, e.e, "DELETE", "/api/v1/learning/path/history?language=en", e.session, nil); code != http.StatusOK {
		t.Fatalf("clear: %d %v", code, body)
	}

	if code, body := chooseLevel(t, e, "en", "A1"); code != http.StatusBadRequest {
		t.Fatalf("picking after the history is cleared: %d %v", code, body)
	}
}
