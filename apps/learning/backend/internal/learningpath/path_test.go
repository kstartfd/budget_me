package learningpath_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningpath/placement"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

const authSecret = "learning-integration-test-secret-long-enough"

// stubProvider stands in for the three agent providers. GETs answer the key
// check; POSTs are completions answered from a queue (or the default text)
// and the request prompts are remembered, so a test sees what the server sent.
type stubProvider struct {
	mu          sync.Mutex
	defaultText string
	queue       []string
	prompts     []string
}

func (p *stubProvider) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if r.Method != http.MethodPost {
		_, _ = w.Write([]byte(`{"data":[]}`))
		return
	}

	raw, _ := io.ReadAll(r.Body)

	p.mu.Lock()
	text := p.defaultText
	if len(p.queue) > 0 {
		text = p.queue[0]
		p.queue = p.queue[1:]
	}
	p.prompts = append(p.prompts, string(raw))
	p.mu.Unlock()

	switch {
	case strings.HasSuffix(r.URL.Path, "/v1/messages"):
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":` + quote(text) + `}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	case strings.HasSuffix(r.URL.Path, "/responses"):
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":` + quote(text) + `}]}]}`))
	default:
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + quote(text) + `}}]}`))
	}
}

func (p *stubProvider) enqueue(texts ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append(p.queue, texts...)
}

func (p *stubProvider) setDefault(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultText = text
}

func (p *stubProvider) requests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.prompts...)
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

type env struct {
	e        *echo.Echo
	q        *db.Queries
	pool     *pgxpool.Pool
	user     uuid.UUID
	session  string
	provider *stubProvider
}

func setup(t *testing.T) env {
	t.Helper()
	return setupWith(t, "")
}

func setupWith(t *testing.T, defaultReply string) env {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)
	raw := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	provider := &stubProvider{defaultText: defaultReply}
	providers := httptest.NewServer(http.HandlerFunc(provider.serve))
	t.Cleanup(providers.Close)

	cfg := config.Config{AuthSecret: authSecret, LLMURLs: llm.URLs{Anthropic: providers.URL, OpenAI: providers.URL, DeepSeek: providers.URL}}

	return env{e: server.New(cfg, pool, testdb.AccountPool(t)), q: q, pool: pool, user: user, session: raw, provider: provider}
}

func request(t *testing.T, e *echo.Echo, method, path, raw string, body any) (int, map[string]any) {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+raw)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("%s %s: %s", method, path, rec.Body.String())
	}

	return rec.Code, response
}

func data(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	d, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected response: %v", response)
	}
	return d
}

// listPayload is the data envelope of an endpoint whose payload is a list.
func listPayload(t *testing.T, response map[string]any) []map[string]any {
	t.Helper()
	items, ok := response["data"].([]any)
	if !ok {
		t.Fatalf("unexpected response: %v", response)
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		out[i], ok = item.(map[string]any)
		if !ok {
			t.Fatalf("unexpected item: %v", item)
		}
	}
	return out
}

// chooseAgent saves a working agent choice against the stub providers.
func chooseAgent(t *testing.T, e env) {
	t.Helper()
	code, body := request(t, e.e, "PUT", "/api/v1/learning/ai/settings", e.session, map[string]any{
		"provider": "anthropic", "model": "claude-opus-5-5", "level": "high", "apiKey": "sk-ant-test-key-1234",
	})
	if code != http.StatusOK {
		t.Fatalf("choose agent: %d %v", code, body)
	}
}

// startPlacement opens a placement test and returns its id and first batch.
func startPlacement(t *testing.T, e env) (string, []any) {
	t.Helper()
	id, state := startPlacementState(t, e)
	return id, state["questions"].([]any)
}

// startPlacementState opens a placement test and returns its id and whole DTO,
// level included.
func startPlacementState(t *testing.T, e env) (string, map[string]any) {
	t.Helper()
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{
		"language": "en", "translationLanguage": "ru",
	})
	if code != http.StatusCreated {
		t.Fatalf("start placement: %d %v", code, body)
	}
	d := data(t, body)
	questions, _ := d["questions"].([]any)
	if len(questions) == 0 {
		t.Fatalf("no questions in the batch: %v", d)
	}
	return d["id"].(string), d
}

// answerBatch posts one answer per question; right=false picks a wrong option.
// It returns the test DTO after the batch.
func answerBatch(t *testing.T, e env, testID string, questions []any, right bool) map[string]any {
	t.Helper()

	answers := make([]map[string]any, len(questions))
	for i, item := range questions {
		q := item.(map[string]any)
		key := q["key"].(string)
		picked := ""
		if right {
			question, _, ok := placement.Find("en", key)
			if !ok {
				t.Fatalf("unknown question %q", key)
			}
			picked = question.Answer
		} else {
			for _, opt := range q["options"].([]any) {
				question, _, _ := placement.Find("en", key)
				if opt.(string) != question.Answer {
					picked = opt.(string)
					break
				}
			}
		}
		answers[i] = map[string]any{"key": key, "answer": picked}
	}

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/answers", e.session, map[string]any{"answers": answers})
	if code != http.StatusOK {
		t.Fatalf("answers: %d %v", code, body)
	}

	return data(t, body)
}

// climb finishes a placement at wantReached: batches are answered right up to
// the level above it, and that level's batch is failed on purpose — failing a
// level lands the result one rung under it.
func climb(t *testing.T, e env, wantReached string) string {
	t.Helper()

	testID, state := startPlacementState(t, e)
	failAt := aboveLevel(wantReached)

	for {
		right := state["level"].(string) != failAt
		state = answerBatch(t, e, testID, state["questions"].([]any), right)
		if state["status"] == "done" {
			if state["levelReached"] != wantReached {
				t.Fatalf("reached %v, want %s", state["levelReached"], wantReached)
			}
			return testID
		}
	}
}

// aboveLevel is the rung over, named by where failing IT lands: failing A2
// places the learner at A1. C2 has none — climbing to it never fails.
func aboveLevel(reached string) string {
	if next, ok := placement.Next(placement.Level(reached)); ok {
		return string(next)
	}

	return ""
}

const programReply = `{"summary":"Solid A1 with clear A2 gaps.","topics":[
	{"skill":"grammar","title":"Past simple of regular verbs","goal":"Use the past simple of regular verbs in short sentences about yesterday."},
	{"skill":"vocabulary","title":"Food and drink words","goal":"Name twenty common foods and drinks."},
	{"skill":"reading","title":"Short everyday notices","goal":"Understand opening hours and prices in short notices."},
	{"skill":"grammar","title":"Present perfect with since and for","goal":"Say how long something has been true with since and for."}]}`

func analyze(t *testing.T, e env, testID string) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/analyze", e.session, nil)
	if code != http.StatusCreated {
		t.Fatalf("analyze: %d %v", code, body)
	}
	return data(t, body)
}

// TestPlacementBuildsAProgram walks the whole front half: test, climb, fail,
// analyze, program saved with topics.
func TestPlacementBuildsAProgram(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	// Fail after A1: answer A1 right, then A2 wrong.
	testID, questions := startPlacement(t, e)
	state := answerBatch(t, e, testID, questions, true)
	if state["level"] != "A2" {
		t.Fatalf("the ladder did not climb: %v", state["level"])
	}
	state = answerBatch(t, e, testID, state["questions"].([]any), false)
	if state["status"] != "done" || state["levelReached"] != "A1" {
		t.Fatalf("the climb must land under the failed level: %v", state)
	}

	path := analyze(t, e, testID)
	if path["level"] != "A1" || path["status"] != "active" {
		t.Fatalf("path: %v", path)
	}
	topics := path["topics"].([]any)
	if len(topics) != 4 {
		t.Fatalf("topics: %v", topics)
	}
	first := topics[0].(map[string]any)
	if first["status"] != "not_started" || first["position"].(float64) != 0 {
		t.Fatalf("first topic: %v", first)
	}

	// The analyst must have seen the wrong answers and none of the answers.
	prompts := e.provider.requests()
	if len(prompts) != 1 {
		t.Fatalf("agent calls: %d", len(prompts))
	}
	if !strings.Contains(prompts[0], "wrongAnswers") || !strings.Contains(prompts[0], "levelReached") {
		t.Fatalf("the analyst did not get the test result:\n%s", prompts[0])
	}
}

// TestPlacementAnswersNeverReachTheClient checks the test DTO carries no
// answer keys.
func TestPlacementAnswersNeverReachTheClient(t *testing.T) {
	e := setup(t)

	_, questions := startPlacement(t, e)
	for _, item := range questions {
		q := item.(map[string]any)
		if _, ok := q["answer"]; ok {
			t.Fatalf("the answer left the server: %v", q)
		}
	}
}

// TestAnalyzeRetriesABrokenProgram has the agent answer garbage first; the
// retry must carry the complaint and still produce the program.
func TestAnalyzeRetriesABrokenProgram(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.enqueue("Sure! Here you go:", programReply)

	testID := climb(t, e, "A1")
	path := analyze(t, e, testID)
	if path["level"] != "A1" {
		t.Fatalf("path: %v", path)
	}

	prompts := e.provider.requests()
	if len(prompts) != 2 {
		t.Fatalf("calls: %d", len(prompts))
	}
	if !strings.Contains(prompts[1], "could not be used") {
		t.Fatalf("the retry did not say why:\n%s", prompts[1])
	}
}

// TestAnalyzeWithoutAnAgent is the setup error a learner can act on.
func TestAnalyzeWithoutAnAgent(t *testing.T) {
	e := setup(t)

	testID := climb(t, e, "A1")
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/analyze", e.session, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("analyze with no agent: %d %v", code, body)
	}
}

const exercisesReply = `{"exercises":[
	{"kind":"choice","text":"","prompt":"She ___ to school every day.","options":["go","goes","going","went"],"answer":"goes"},
	{"kind":"fill","text":"","prompt":"Yesterday I ... my bike. (ride)","options":[],"answer":"rode"},
	{"kind":"translate","text":"","prompt":"Translate into English: Я живу в Москве.","options":[],"answer":"I live in Moscow."}]}`

// generateExercises opens a topic's exercise set.
func generateExercises(t *testing.T, e env, topicID string) []any {
	t.Helper()
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("generate: %d %v", code, body)
	}
	list := listPayload(t, body)
	if len(list) == 0 {
		t.Fatalf("no exercises")
	}
	return toAny(list)
}

func toAny(list []map[string]any) []any {
	out := make([]any, len(list))
	for i, item := range list {
		out[i] = item
	}
	return out
}

// TestTopicExercisesAndAttempts covers generation, caching, grading, the
// agent's explanations and completing the topic.
func TestTopicExercisesAndAttempts(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topic := path["topics"].([]any)[0].(map[string]any)
	topicID := topic["id"].(string)

	e.provider.enqueue(exercisesReply)
	exercises := generateExercises(t, e, topicID)

	// Reopening the topic hands back the same set, with no new agent call.
	before := len(e.provider.requests())
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("cached generate: %d %v", code, body)
	}
	if again := listPayload(t, body); len(again) != len(exercises) {
		t.Fatalf("the cached set changed: %v", again)
	}
	if len(e.provider.requests()) != before {
		t.Fatal("a cache hit still spent the agent")
	}

	// The topic is in progress now.
	detail := topicDetail(t, e, topicID)
	if detail["topic"].(map[string]any)["status"] != "in_progress" {
		t.Fatalf("topic status: %v", detail["topic"])
	}

	// A right choice answer is graded on the spot: no agent call, no feedback.
	choice := exercises[0].(map[string]any)
	if _, ok := choice["answer"]; ok {
		t.Fatalf("the exercise answer left the server: %v", choice)
	}
	calls := len(e.provider.requests())
	attempt(t, e, choice["id"].(string), "goes", true, "")
	if len(e.provider.requests()) != calls {
		t.Fatal("a right choice answer spent the agent")
	}

	// A wrong fill answer is graded locally and explained by the agent.
	fill := exercises[1].(map[string]any)
	e.provider.enqueue(`{"correct":false,"feedback":"'Ride' in the past is 'rode'."}`)
	attempt(t, e, fill["id"].(string), "rided", false, "rode")

	// A translation is graded by the agent.
	translate := exercises[2].(map[string]any)
	e.provider.enqueue(`{"correct":true,"feedback":""}`)
	attempt(t, e, translate["id"].(string), "I live in Moscow.", true, "")

	// Completing the topic sticks.
	code, body = request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/complete", e.session, nil)
	if code != http.StatusOK || data(t, body)["status"] != "completed" {
		t.Fatalf("complete: %d %v", code, body)
	}

	// A completed topic rejects new work.
	code, _ = request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises", e.session, nil)
	if code != http.StatusConflict {
		t.Fatalf("generate on a completed topic: %d", code)
	}
}

func attempt(t *testing.T, e env, exerciseID, answer string, wantCorrect bool, wantFeedback string) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/exercises/"+exerciseID+"/attempt", e.session, map[string]any{"answer": answer})
	if code != http.StatusOK {
		t.Fatalf("attempt: %d %v", code, body)
	}
	d := data(t, body)
	if d["correct"] != wantCorrect {
		t.Fatalf("correct: %v, want %v (%v)", d["correct"], wantCorrect, d)
	}
	if wantFeedback != "" && !strings.Contains(d["feedback"].(string), wantFeedback) {
		t.Fatalf("feedback: %v", d["feedback"])
	}
	return d
}

func topicDetail(t *testing.T, e env, topicID string) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "GET", "/api/v1/learning/path/topics/"+topicID, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("topic detail: %d %v", code, body)
	}
	return data(t, body)
}

// TestFinalTestFinishesThePath: passing the final test archives the program as
// finished and the history keeps it; failing leaves it running.
func TestFinalTestFinishesThePath(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A1"))

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/final-tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated {
		t.Fatalf("start final: %d %v", code, body)
	}
	testID := data(t, body)["id"].(string)

	state := answerBatch(t, e, testID, data(t, body)["questions"].([]any), true)
	if state["status"] != "done" || state["passed"] != true {
		t.Fatalf("final: %v", state)
	}

	code, body = request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
	if code != http.StatusNotFound {
		t.Fatalf("a finished path must not be active: %d %v", code, body)
	}

	code, body = request(t, e.e, "GET", "/api/v1/learning/path/history", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("history: %d %v", code, body)
	}
	paths := listPayload(t, body)
	if len(paths) == 0 {
		t.Fatal("history is empty")
	}
	if paths[0]["status"] != "finished" {
		t.Fatalf("newest history entry: %v", paths[0])
	}
}

// TestResetArchivesTheProgram: reset closes the active program into the
// history and frees the language for a fresh placement.
func TestResetArchivesTheProgram(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "B1"))

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"})
	if code != http.StatusOK {
		t.Fatalf("reset: %d %v", code, body)
	}

	// The language is free again.
	startPlacement(t, e)

	code, body = request(t, e.e, "GET", "/api/v1/learning/path/history", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("history: %d %v", code, body)
	}
	paths := listPayload(t, body)
	if len(paths) == 0 {
		t.Fatal("history is empty")
	}
	if paths[0]["status"] != "abandoned" {
		t.Fatalf("the reset program must sit in the history: %v", paths[0])
	}
}

// TestSecondProgramIsASeedsTheSourceLabel — the analyzer labels a re-analysis
// as a reset in the history.
func TestSecondProgramIsMarkedAsReset(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A1"))

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"})
	if code != http.StatusOK {
		t.Fatalf("reset: %d %v", code, body)
	}

	analyze(t, e, climb(t, e, "A2"))

	// The re-analysis is the active program now; the history only holds closed
	// ones, where the first program sits marked as a placement.
	code, body = request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("active path: %d %v", code, body)
	}
	if data(t, body)["source"] != "reset" {
		t.Fatalf("the re-analysis must be marked as a reset: %v", data(t, body))
	}
}

// TestOpenTestIsHandedBack: starting a placement twice returns the open test
// instead of splitting it.
func TestOpenTestIsHandedBack(t *testing.T) {
	e := setup(t)

	first, _ := startPlacement(t, e)
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated || data(t, body)["id"] != first {
		t.Fatalf("a second start must hand back the open test: %d %v", code, body)
	}
}

// TestUnknownAnswerKeyIsRefused keeps the batch protocol honest.
func TestUnknownAnswerKeyIsRefused(t *testing.T) {
	e := setup(t)

	testID, _ := startPlacement(t, e)
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/answers", e.session, map[string]any{
		"answers": []map[string]any{{"key": "en/C2/g1", "answer": "would"}},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("a key of another level must be refused: %d %v", code, body)
	}
}

// TestRegenerateKeepsAttemptHistory: new exercises replace the active set,
// while attempts on earlier sets remain available in the topic history.
func TestRegenerateKeepsAttemptHistory(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.enqueue(exercisesReply)
	generateExercises(t, e, topicID)

	// No attempts yet: regenerating works.
	e.provider.enqueue(exercisesReply)
	code, _ := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises?regenerate=1", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("regenerate before attempts: %d", code)
	}

	// One attempt is retained when a new set is generated.
	e.provider.enqueue(`{"correct":false,"feedback":"no"}`)
	exerciseID := generateExercises(t, e, topicID)[0].(map[string]any)["id"].(string)
	attempt(t, e, exerciseID, "wrong answer here", false, "")
	e.provider.enqueue(exercisesReply)
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises?regenerate=1", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("regenerate after attempts: %d %v", code, body)
	}
	newSet := listPayload(t, body)
	if len(newSet) == 0 || newSet[0].(map[string]any)["id"] == exerciseID {
		t.Fatalf("regeneration did not replace the current set: %v", newSet)
	}
	detail := topicDetail(t, e, topicID)
	if history := detail["attempts"].([]any); len(history) != 1 || history[0].(map[string]any)["exerciseId"] != exerciseID {
		t.Fatalf("past attempt was lost: %v", history)
	}
}

// TestClimbToTheTop: passing C2 ends the ladder at C2.
func TestClimbToTheTop(t *testing.T) {
	e := setup(t)

	// The whole ladder right: A1..C2 then done at C2.
	testID, questions := startPlacement(t, e)
	climbs := 0
	for {
		state := answerBatch(t, e, testID, questions, true)
		if state["status"] == "done" {
			if state["levelReached"] != "C2" {
				t.Fatalf("top of the ladder: %v", state["levelReached"])
			}
			break
		}
		questions = state["questions"].([]any)
		climbs++
		if climbs > len(placement.Levels) {
			t.Fatal("the ladder never ends")
		}
	}
	_ = fmt.Sprint(climbs)
}
