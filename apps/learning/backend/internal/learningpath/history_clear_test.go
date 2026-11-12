package learningpath_test

import (
	"net/http"
	"testing"
)

// TestTheHistoryCanBeCleared: closed programs of a language go with everything
// under them, the running program stays, and the language is required. What
// the history held is what let a learner start above A1, so that is gone too.
func TestTheHistoryCanBeCleared(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	const history = "/api/v1/learning/path/history"

	if code, _ := request(t, e.e, "DELETE", history, e.session, nil); code != http.StatusBadRequest {
		t.Fatalf("clearing without a language: %d", code)
	}

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	// The running program is not history: nothing goes.
	code, body := request(t, e.e, "DELETE", history+"?language=en", e.session, nil)
	if code != http.StatusOK || data(t, body)["deleted"] != float64(0) {
		t.Fatalf("clearing with only a running program: %d %v", code, body)
	}

	if topicDetail(t, e, topicID)["topic"] == nil {
		t.Fatal("the running program lost its topic")
	}

	// A passed final test closes the program as finished: it is in the history, and
	// it is what would let the next test start at A2 (see the test of the ladder).
	code, body = request(t, e.e, "POST", "/api/v1/learning/path/final-tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated {
		t.Fatalf("start final: %d %v", code, body)
	}

	if state := answerBatch(t, e, data(t, body)["id"].(string), data(t, body)["questions"].([]any), true); state["passed"] != true {
		t.Fatalf("final: %v", state)
	}

	start := func() int {
		code, _ := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru", "level": "A2"})
		return code
	}

	code, body = request(t, e.e, "GET", history, e.session, nil)
	if code != http.StatusOK || len(listPayload(t, body)) != 1 {
		t.Fatalf("history before clearing: %d %v", code, body)
	}

	// Another language's history is not touched.
	if code, body := request(t, e.e, "DELETE", history+"?language=pl", e.session, nil); code != http.StatusOK || data(t, body)["deleted"] != float64(0) {
		t.Fatalf("clearing another language: %d %v", code, body)
	}

	code, body = request(t, e.e, "DELETE", history+"?language=en", e.session, nil)
	if code != http.StatusOK || data(t, body)["deleted"] != float64(1) {
		t.Fatalf("clearing: %d %v", code, body)
	}

	code, body = request(t, e.e, "GET", history, e.session, nil)
	if code != http.StatusOK || len(listPayload(t, body)) != 0 {
		t.Fatalf("history after clearing: %d %v", code, body)
	}

	var topics int
	if err := e.pool.QueryRow(t.Context(), "SELECT count(*) FROM learning_path_topics WHERE id = $1", topicID).Scan(&topics); err != nil {
		t.Fatal(err)
	}

	if topics != 0 {
		t.Fatal("a cleared program must take its topics with it")
	}

	// The proof of the confirmed level went with it: the test starts from the bottom again.
	if code := start(); code != http.StatusBadRequest {
		t.Fatalf("above A1 after clearing the history: %d", code)
	}
}
