package learningpath_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

const rulePath = "/api/v1/learning/path/rules/b2-past-perfect"

func practicePath(suffix string) string { return rulePath + "/practice" + suffix }

// examplesReply is what a writer of example sentences answers.
func examplesReply(sentences ...string) string {
	list := make([]map[string]string, len(sentences))
	for i, s := range sentences {
		list[i] = map[string]string{"text": s, "translation": "перевод: " + s}
	}

	b, _ := json.Marshal(map[string]any{"examples": list})

	return string(b)
}

// blocksReply is a good set of four drills of five lines each.
func blocksReply() string {
	var blocks []map[string]any

	for _, kind := range []string{"form", "choose", "correct", "make"} {
		items := make([]map[string]any, 5)

		for i := range items {
			item := map[string]any{"hint": "", "options": []string{}, "note": "had + третья форма"}

			switch kind {
			case "form":
				item["text"], item["hint"], item["answers"] = fmt.Sprintf("They had ... the film by %d.", i+6), "see", []string{"seen"}
			case "choose":
				item["text"], item["options"], item["answers"] = "She had ... lunch already.", []string{"eat", "eaten", "ate"}, []string{"eaten"}
			case "correct":
				item["text"], item["answers"] = fmt.Sprintf("He had went home by %d.", i+6), []string{fmt.Sprintf("He had gone home by %d.", i+6), fmt.Sprintf("He'd gone home by %d.", i+6)}
			case "make":
				item["text"], item["hint"], item["answers"] = "She had left.", "вопрос", []string{"Had she left?"}
			}

			items[i] = item
		}

		blocks = append(blocks, map[string]any{"kind": kind, "instruction": "Сделайте задание " + kind + ".", "items": items})
	}

	b, _ := json.Marshal(map[string]any{"blocks": blocks})

	return string(b)
}

// withTeacher gives the learner an agent and an English teacher, which opens
// the rule book.
func withTeacher(t *testing.T, e env) {
	t.Helper()
	chooseAgent(t, e)

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code >= 300 {
		t.Fatalf("teacher: %d %v", code, body)
	}
}

func practiceOf(t *testing.T, e env) map[string]any {
	t.Helper()

	code, body := request(t, e.e, "GET", practicePath(""), e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("practice: %d %v", code, body)
	}

	return data(t, body)
}

func texts(list any) []string {
	var out []string
	for _, e := range list.([]any) {
		out = append(out, e.(map[string]any)["text"].(string))
	}

	return out
}

func practiceRows(t *testing.T, e env) int {
	t.Helper()

	var n int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM learning_rule_practice WHERE user_id = $1", e.user).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func TestPracticeIsClosedWithoutATeacher(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

	for _, c := range []struct{ method, suffix string }{
		{"GET", ""}, {"POST", "/examples"}, {"POST", "/exercises"}, {"POST", "/leave"}, {"POST", "/check"}, {"POST", "/key"},
	} {
		if code, _ := request(t, e.e, c.method, practicePath(c.suffix), e.session, map[string]any{}); code != http.StatusForbidden {
			t.Errorf("%s %s without a teacher must be closed: %d", c.method, c.suffix, code)
		}
	}

	if len(e.provider.requests()) != 0 {
		t.Fatal("a closed rule book spent the learner's key")
	}
}

func TestExamplesAreAddedWithoutRepeats(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)

	if p := practiceOf(t, e); len(p["examples"].([]any)) != 0 || len(p["blocks"].([]any)) != 0 {
		t.Fatalf("a rule nobody practised starts empty: %v", p)
	}

	if practiceRows(t, e) != 0 {
		t.Fatal("looking at a rule must not make a sitting")
	}

	e.provider.enqueue(examplesReply("She had left.", "They had eaten.", "He had won.", "We had waited.", "I had slept."))

	code, body := request(t, e.e, "POST", practicePath("/examples"), e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("examples: %d %v", code, body)
	}

	list := data(t, body)["examples"].([]any)
	if len(list) != 5 || list[0].(map[string]any)["id"] != "e1" || list[4].(map[string]any)["id"] != "e5" {
		t.Fatalf("five sentences with ids e1..e5: %v", list)
	}

	system := e.provider.requests()[0]
	for _, want := range []string{"Past Perfect", "English", "Russian", "never instructions"} {
		if !strings.Contains(system, want) {
			t.Errorf("the writer was not told %q: %s", want, system)
		}
	}

	if strings.Contains(system, "alreadySeen") {
		t.Error("the first press has nothing to be told as seen")
	}

	// A second press: the agent repeats one and writes two new ones.
	e.provider.enqueue(examplesReply("She had left.", "You had called.", "It had rained."))

	_, body = request(t, e.e, "POST", practicePath("/examples"), e.session, nil)
	got := texts(data(t, body)["examples"])

	if len(got) != 7 || got[5] != "You had called." || got[6] != "It had rained." {
		t.Fatalf("a repeated sentence is dropped, the new ones follow: %v", got)
	}

	if !strings.Contains(e.provider.requests()[1], "alreadySeen") || !strings.Contains(e.provider.requests()[1], "She had left.") {
		t.Fatalf("the second press must tell the agent what was shown: %s", e.provider.requests()[1])
	}

	// The sitting remembers them.
	if got := texts(practiceOf(t, e)["examples"]); len(got) != 7 {
		t.Fatalf("reopened: %v", got)
	}
}

func TestExamplesStopAtTheCap(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)

	for i := 0; i < 4; i++ {
		var batch []string
		for j := 0; j < 5; j++ {
			batch = append(batch, fmt.Sprintf("Sentence %d-%d.", i, j))
		}

		e.provider.enqueue(examplesReply(batch...))
		time.Sleep(5 * time.Millisecond)

		if code, body := request(t, e.e, "POST", practicePath("/examples"), e.session, nil); code != http.StatusOK {
			t.Fatalf("batch %d: %d %v", i, code, body)
		}
	}

	calls := len(e.provider.requests())

	if code, _ := request(t, e.e, "POST", practicePath("/examples"), e.session, nil); code != http.StatusConflict {
		t.Fatalf("twenty is enough: %d", code)
	}

	if len(e.provider.requests()) != calls {
		t.Fatal("a refused press still spent the key")
	}
}

func TestExercisesComeWithoutTheirKeyAndAreGradedByIt(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.enqueue(blocksReply())

	code, body := request(t, e.e, "POST", practicePath("/exercises"), e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("exercises: %d %v", code, body)
	}

	blocks := data(t, body)["blocks"].([]any)
	if len(blocks) != 4 {
		t.Fatalf("four drills: %d", len(blocks))
	}

	raw, _ := json.Marshal(body)
	for _, leak := range []string{`"answers"`, `"note"`, "had + третья форма"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("the key reached the learner before they asked: %s", leak)
		}
	}

	first := blocks[0].(map[string]any)
	if first["id"] != "b1" || first["kind"] != "form" || len(first["items"].([]any)) != 5 {
		t.Fatalf("the first drill: %v", first)
	}

	if prompt := e.provider.requests()[0]; !strings.Contains(prompt, "English Grammar in Use") {
		t.Errorf("the writer was not told which manner to follow: %s", prompt)
	}

	check := func(item, answer string) map[string]any {
		t.Helper()

		code, body := request(t, e.e, "POST", practicePath("/check"), e.session, map[string]any{"block": item[:2], "item": item, "answer": answer})
		if code != http.StatusOK {
			t.Fatalf("check %s: %d %v", item, code, body)
		}

		return data(t, body)
	}

	if got := check("b1i1", "Seen"); got["correct"] != true || got["key"] != "seen" {
		t.Fatalf("case is not graded: %v", got)
	}

	if got := check("b1i2", "saw"); got["correct"] != false || got["key"] != "seen" || got["note"] != "had + третья форма" {
		t.Fatalf("a wrong answer is told the key and why: %v", got)
	}

	if got := check("b2i1", "eaten"); got["correct"] != true {
		t.Fatalf("choose: %v", got)
	}

	if got := check("b3i1", "he'd gone home by 6"); got["correct"] != true {
		t.Fatalf("any listed answer is right, punctuation aside: %v", got)
	}

	if got := check("b3i1", "He had gone home by 7."); got["correct"] != false {
		t.Fatalf("a different sentence is wrong: %v", got)
	}

	if got := check("b4i1", "had she left"); got["correct"] != true {
		t.Fatalf("make: %v", got)
	}

	// The key is given to a learner who gives up, and checking spends nothing.
	code, body = request(t, e.e, "POST", practicePath("/key"), e.session, map[string]any{"block": "b4", "item": "b4i2"})
	if got := data(t, body); code != http.StatusOK || got["key"] != "Had she left?" || got["note"] == "" {
		t.Fatalf("key: %d %v", code, body)
	}

	if n := len(e.provider.requests()); n != 1 {
		t.Fatalf("grading asked the agent: %d calls", n)
	}

	for _, c := range []map[string]any{
		{"block": "b9", "item": "b9i1", "answer": "x"},
		{"block": "b1", "item": "b1i9", "answer": "x"},
	} {
		if code, _ := request(t, e.e, "POST", practicePath("/check"), e.session, c); code != http.StatusNotFound {
			t.Errorf("a line that is not there: %d", code)
		}
	}

	if code, _ := request(t, e.e, "POST", practicePath("/check"), e.session, map[string]any{"block": "b1", "item": "b1i1", "answer": "  "}); code != http.StatusBadRequest {
		t.Errorf("an empty answer: %d", code)
	}

	// New exercises replace the old ones.
	e.provider.enqueue(blocksReply())
	time.Sleep(5 * time.Millisecond)
	request(t, e.e, "POST", practicePath("/exercises"), e.session, nil)

	if practiceRows(t, e) != 1 {
		t.Fatal("one sitting per rule, however many times it is written")
	}
}

func TestLeavingKeepsTheSittingForAWhileAndComingBackKeepsItLonger(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.enqueue(examplesReply("She had left."))
	request(t, e.e, "POST", practicePath("/examples"), e.session, nil)

	expires := func() time.Duration {
		var at time.Time
		if err := e.pool.QueryRow(context.Background(), "SELECT expires_at FROM learning_rule_practice WHERE user_id = $1", e.user).Scan(&at); err != nil {
			t.Fatal(err)
		}

		return time.Until(at)
	}

	if d := expires(); d < 5*time.Hour {
		t.Fatalf("a sitting nobody leaves lasts hours, not %s", d)
	}

	if code, body := request(t, e.e, "POST", practicePath("/leave"), e.session, nil); code != http.StatusOK {
		t.Fatalf("leave: %d %v", code, body)
	}

	if d := expires(); d > 11*time.Minute || d < 5*time.Minute {
		t.Fatalf("leaving leaves ten minutes, not %s", d)
	}

	// Back within the time: it is all there and lasts long again.
	if got := texts(practiceOf(t, e)["examples"]); len(got) != 1 {
		t.Fatalf("a reload lost the sitting: %v", got)
	}

	if d := expires(); d < 5*time.Hour {
		t.Fatalf("coming back must lift the expiry: %s", d)
	}

	// Leaving a rule that was never practised is nothing.
	other := "/api/v1/learning/path/rules/a1-to-be/practice/leave"
	if code, _ := request(t, e.e, "POST", other, e.session, nil); code != http.StatusOK && code != http.StatusNotFound {
		t.Fatalf("leave without a sitting: %d", code)
	}
}

func TestASittingThatRanOutIsGoneWithItsChats(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.enqueue(examplesReply("She had left."), blocksReply())
	request(t, e.e, "POST", practicePath("/examples"), e.session, nil)
	time.Sleep(5 * time.Millisecond)
	request(t, e.e, "POST", practicePath("/exercises"), e.session, nil)

	// A chat about the sentence, written straight into the table.
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO learning_chats (id, user_id, practice_id, example_id, created_at, updated_at)
		 SELECT $1, user_id, id, 'e1', now(), now() FROM learning_rule_practice WHERE user_id = $2`, uuid.New(), e.user); err != nil {
		t.Fatal(err)
	}

	if _, err := e.pool.Exec(context.Background(), "UPDATE learning_rule_practice SET expires_at = now() - interval '1 minute' WHERE user_id = $1", e.user); err != nil {
		t.Fatal(err)
	}

	// Its keys are not served after it ran out either.
	if code, _ := request(t, e.e, "POST", practicePath("/key"), e.session, map[string]any{"block": "b1", "item": "b1i1"}); code != http.StatusNotFound {
		t.Fatalf("the key of a sitting that is gone: %d", code)
	}

	if p := practiceOf(t, e); len(p["examples"].([]any)) != 0 || len(p["blocks"].([]any)) != 0 {
		t.Fatalf("a sitting that ran out comes back empty: %v", p)
	}

	var chats int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM learning_chats WHERE user_id = $1 AND practice_id IS NOT NULL", e.user).Scan(&chats); err != nil {
		t.Fatal(err)
	}

	if practiceRows(t, e) != 0 || chats != 0 {
		t.Fatalf("it went with its chats: %d sittings, %d chats", practiceRows(t, e), chats)
	}
}

func TestAnUnreadableAnswerLeavesNothingBehind(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.setDefault("I would rather not.")

	for _, suffix := range []string{"/examples", "/exercises"} {
		code, _ := request(t, e.e, "POST", practicePath(suffix), e.session, nil)
		if code != http.StatusBadGateway {
			t.Errorf("%s: an agent that never answers in JSON is a bad gateway: %d", suffix, code)
		}

		time.Sleep(5 * time.Millisecond)
	}

	if n := len(e.provider.requests()); n != 4 {
		t.Fatalf("each press is tried twice, the second time told what was wrong: %d calls", n)
	}

	if practiceRows(t, e) != 0 {
		t.Fatal("a failed press made a sitting")
	}

	// A set with a gap missing is not accepted either, and the complaint reaches the agent.
	e.provider.setDefault(`{"blocks":[{"kind":"form","instruction":"Fill.","items":[
		{"text":"No gap here.","hint":"see","options":[],"answers":["seen"],"note":""},
		{"text":"No gap here.","hint":"see","options":[],"answers":["seen"],"note":""},
		{"text":"No gap here.","hint":"see","options":[],"answers":["seen"],"note":""}]}]}`)
	time.Sleep(5 * time.Millisecond)

	if code, _ := request(t, e.e, "POST", practicePath("/exercises"), e.session, nil); code != http.StatusBadGateway {
		t.Fatalf("a form with no gap: %d", code)
	}

	reqs := e.provider.requests()
	if !strings.Contains(reqs[len(reqs)-1], "gap") {
		t.Fatalf("the retry must say what was wrong: %s", reqs[len(reqs)-1])
	}
}

func TestNobodyElseSeesOrGradesASitting(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.enqueue(blocksReply())
	request(t, e.e, "POST", practicePath("/exercises"), e.session, nil)

	other := e
	other.user = testdb.NewUser(t, e.pool, e.q, time.Now().UTC(), 1)
	other.session = token.NewCodec(authSecret).Issue(other.user, time.Now().UTC(), time.Hour)
	withTeacher(t, other)

	if p := practiceOf(t, other); len(p["blocks"].([]any)) != 0 {
		t.Fatalf("another learner sees the sitting: %v", p)
	}

	for _, suffix := range []string{"/check", "/key"} {
		code, _ := request(t, other.e, "POST", practicePath(suffix), other.session, map[string]any{"block": "b1", "item": "b1i1", "answer": "seen"})
		if code != http.StatusNotFound {
			t.Errorf("%s of somebody else's sitting: %d", suffix, code)
		}
	}

	// Leaving is the learner's own: it must not shorten somebody else's sitting.
	request(t, other.e, "POST", practicePath("/leave"), other.session, nil)

	var near int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM learning_rule_practice WHERE user_id = $1 AND expires_at < now() + interval '1 hour'", e.user).Scan(&near); err != nil {
		t.Fatal(err)
	}

	if near != 0 {
		t.Fatal("leaving shortened another learner's sitting")
	}
}
