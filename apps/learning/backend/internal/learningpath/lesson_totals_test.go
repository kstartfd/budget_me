package learningpath_test

import (
	"net/http"
	"testing"
)

const totalsPath = "/api/v1/learning/path/lessons/totals"

// TestLessonTotalsAreServedInSeconds: nothing studied is zero in all three
// windows, a zone the server does not know only moves midnight, and a running
// lesson already counts.
func TestLessonTotalsAreServedInSeconds(t *testing.T) {
	e := setup(t)

	totals := func(query string) map[string]any {
		t.Helper()

		code, body := request(t, e.e, "GET", totalsPath+query, e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("totals%s: %d %v", query, code, body)
		}

		return data(t, body)
	}

	for _, query := range []string{"", "?tz=Europe/Moscow", "?tz=Mars/Olympus"} {
		got := totals(query)
		if got["today"] != float64(0) || got["week"] != float64(0) || got["month"] != float64(0) {
			t.Fatalf("nothing studied yet%s: %v", query, got)
		}
	}

	lessonCall(t, e, "POST", "lessons", http.StatusOK)

	got := totals("?tz=Europe/Moscow")
	today, week, month := got["today"].(float64), got["week"].(float64), got["month"].(float64)

	if today < 0 || today > 5 || week < today || month < week {
		t.Fatalf("a lesson just started: %v", got)
	}

	if code, _ := request(t, e.e, "GET", totalsPath, "", nil); code != http.StatusUnauthorized {
		t.Fatalf("without a session: %d", code)
	}
}
