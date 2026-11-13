package learningpath_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestExamplesInTheTheoryAreMarkedWhateverTheLanguage: the page sets what is
// between backticks in a font of its own, so the library marks its examples,
// and a program saved before the marks were added shows them all the same.
func TestExamplesInTheTheoryAreMarkedWhateverTheLanguage(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	id := teacherID(t, e)
	analyze(t, e, climb(t, e, "A1"))

	theory := func() string {
		code, body := request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("path: %d %v", code, body)
		}

		return data(t, body)["topics"].([]any)[0].(map[string]any)["theory"].(string)
	}

	// Those rows were saved with the library's text of the time, marks or none.
	if _, err := e.pool.Exec(t.Context(), `update learning_path_topics set theory = 'old text without marks' where source_id <> ''`); err != nil {
		t.Fatal(err)
	}

	if got := theory(); !strings.Contains(got, "`I am, you are, he/she/it is, we/you/they are.`") || !strings.Contains(got, "`I'm from Russia.`") {
		t.Fatalf("the Russian rule must carry its examples in backticks: %q", got)
	}

	explainIn(t, e, id, true)

	if got := theory(); !strings.Contains(got, "`I am, you are, he/she/it is, we/you/they are.`") || !strings.Contains(got, "`She is a doctor.`") {
		t.Fatalf("the English rule must carry its examples in backticks: %q", got)
	}
}
