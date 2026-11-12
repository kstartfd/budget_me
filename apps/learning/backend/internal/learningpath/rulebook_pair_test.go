package learningpath_test

import (
	"net/http"
	"testing"
)

// TestTheRuleBookIsOfThePairAskedFor: a learner who studies English and a
// language whose pair has no rule book gets the English rules on the English
// teacher's screen only. The other teacher's screen gets nothing, not the
// English rules thrown at someone studying Spanish.
func TestTheRuleBookIsOfThePairAskedFor(t *testing.T) {
	e := setup(t)

	for _, language := range []string{"es", "en"} {
		code, body := request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": language, "translationLanguage": "ru"})
		if code >= 300 {
			t.Fatalf("a %s teacher: %d %v", language, code, body)
		}
	}

	const rules = "/api/v1/learning/path/rules"

	if code, _ := request(t, e.e, "GET", rules+"?language=es", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("the Spanish teacher has no rule book, the English one must not stand in for it: %d", code)
	}

	if code, _ := request(t, e.e, "GET", rules+"/b2-past-perfect?language=es", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("an English rule on the Spanish teacher's screen: %d", code)
	}

	if code, _ := request(t, e.e, "GET", rules+"/b2-past-perfect/practice?language=es", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("practice of an English rule on the Spanish teacher's screen: %d", code)
	}

	code, body := request(t, e.e, "GET", rules+"?language=en", e.session, nil)
	if code != http.StatusOK || len(listPayload(t, body)) < 40 {
		t.Fatalf("the English teacher's book: %d %v", code, body)
	}

	if code, _ := request(t, e.e, "GET", rules+"/b2-past-perfect?language=en", e.session, nil); code != http.StatusOK {
		t.Fatalf("an English rule on the English teacher's screen: %d", code)
	}

	// Without a language the book of the first teacher that has one is served, as before.
	if code, _ := request(t, e.e, "GET", rules, e.session, nil); code != http.StatusOK {
		t.Fatalf("the book without a language: %d", code)
	}
}
