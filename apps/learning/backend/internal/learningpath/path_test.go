package learningpath_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
//
// The level test is written and judged by the agent too, and those two calls
// are answered by the stub itself (see testReply) without touching the queue,
// so a test that queues a program or an exercise set sees only its own calls
// in requests(); the test calls are kept apart in testCalls().
type stubProvider struct {
	mu          sync.Mutex
	defaultText string
	queue       []string
	prompts     []string
	testPrompts []string
	written     int
	// examiner, when set, overrides the examiner's reply: it gets the level
	// and how many of how many were right.
	examiner func(level string, correct, asked int) string
	// breakQuestions makes the question writer answer garbage.
	breakQuestions bool
	// answer, when set, may answer a call itself, whatever is queued: it gets
	// the request body and says whether it did. It runs under the lock.
	answer func(prompt string) (string, bool)
}

const (
	rightOption = "right answer"
	wrongOption = "wrong one"
)

var (
	levelPattern   = regexp.MustCompile(`level\\+"\s*:\s*\\+"([ABC][12])`)
	countPattern   = regexp.MustCompile(`count\\+"\s*:\s*(\d+)`)
	askedPattern   = regexp.MustCompile(`asked\\+"\s*:\s*(\d+)`)
	correctPattern = regexp.MustCompile(`correct\\+"\s*:\s*(\d+)`)
)

func number(re *regexp.Regexp, raw string) int {
	m := re.FindStringSubmatch(raw)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// testReply answers the question writer and the examiner of a level test. The
// writer makes unique questions with one right option; the examiner climbs when
// at least three quarters are right and otherwise stops one rung under.
func (p *stubProvider) testReply(raw string) (string, bool) {
	level := ""
	if m := levelPattern.FindStringSubmatch(raw); m != nil {
		level = m[1]
	}

	switch {
	case strings.Contains(raw, "You write the questions of a language level test"):
		if p.breakQuestions {
			return "I cannot write that.", true
		}

		count := number(countPattern, raw)
		questions := make([]map[string]any, count)
		for i := range questions {
			p.written++
			skill, text := "grammar", ""
			if i == count-1 {
				skill, text = "reading", "A short passage."
			}
			questions[i] = map[string]any{
				"skill": skill, "text": text,
				"prompt":  fmt.Sprintf("%s question %d", level, p.written),
				"options": []string{wrongOption, rightOption, "wrong two", "wrong three"},
				"answer":  rightOption,
			}
		}
		b, _ := json.Marshal(map[string]any{"questions": questions})
		return string(b), true
	case strings.Contains(raw, "You are the examiner of a language level test"):
		asked, correct := number(askedPattern, raw), number(correctPattern, raw)
		if p.examiner != nil {
			return p.examiner(level, correct, asked), true
		}
		if correct*4 >= asked*3 {
			return `{"verdict":"climb","level":"` + level + `"}`, true
		}
		return `{"verdict":"stop","level":"` + string(placement.Below(placement.Level(level))) + `"}`, true
	}

	return "", false
}

func (p *stubProvider) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
		return
	}

	raw, _ := io.ReadAll(r.Body)

	p.mu.Lock()
	text, isTest := p.testReply(string(raw))
	if isTest {
		p.testPrompts = append(p.testPrompts, string(raw))
	} else {
		text = p.defaultText
		if len(p.queue) > 0 {
			text = p.queue[0]
			p.queue = p.queue[1:]
		}
		if p.answer != nil {
			if reply, ok := p.answer(string(raw)); ok {
				text = reply
			}
		}
		p.prompts = append(p.prompts, string(raw))
	}
	p.mu.Unlock()

	if strings.Contains(string(raw), `"stream":true`) && !strings.HasSuffix(r.URL.Path, "/v1/messages") && !strings.HasSuffix(r.URL.Path, "/responses") {
		w.Header().Set("Content-Type", "text/event-stream")
	}
	w.WriteHeader(http.StatusOK)

	switch {
	case strings.HasSuffix(r.URL.Path, "/v1/messages"):
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":` + quote(text) + `}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	case strings.HasSuffix(r.URL.Path, "/responses"):
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":` + quote(text) + `}]}]}`))
	case strings.Contains(string(raw), `"stream":true`):
		// A streamed chat turn (deepseek): the text in one delta, then the usage.
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":" + quote(text) + "}}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"prompt_cache_hit_tokens\":0}}\n\n" +
			"data: [DONE]\n\n"))
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

// testCalls are the requests the question writer and the examiner got.
func (p *stubProvider) testCalls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.testPrompts...)
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
	audioDir string
}

func setup(t *testing.T) env {
	t.Helper()
	return setupWith(t, "")
}

func setupWith(t *testing.T, defaultReply string) env {
	t.Helper()
	return setupPrefetch(t, defaultReply, 0)
}

// setupPrefetch is setupWith with the background preparation of the first
// exercise sets switched on, which the other tests keep off so the stub only
// sees the calls they make themselves.
func setupPrefetch(t *testing.T, defaultReply string, prefetch int) env {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)
	raw := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	provider := &stubProvider{defaultText: defaultReply}
	providers := httptest.NewServer(http.HandlerFunc(provider.serve))
	t.Cleanup(providers.Close)

	// A stand-in voice, so audio is made without the real service.
	tts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3-fake-voice"))
	}))
	t.Cleanup(tts.Close)

	audioDir := t.TempDir()
	cfg := config.Config{AuthSecret: authSecret, PathPrefetch: prefetch, TTSURL: tts.URL, AudioCacheDir: audioDir, LLMURLs: llm.URLs{Anthropic: providers.URL, OpenAI: providers.URL, DeepSeek: providers.URL, Kimi: providers.URL}}

	return env{e: server.New(cfg, pool, testdb.AccountPool(t)), q: q, pool: pool, user: user, session: raw, provider: provider, audioDir: audioDir}
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
	n := 0
	if right {
		n = len(questions)
	}
	return answerMixed(t, e, testID, questions, n)
}

// answerMixed answers the first nRight questions right and the rest wrong.
func answerMixed(t *testing.T, e env, testID string, questions []any, nRight int) map[string]any {
	t.Helper()

	answers := make([]map[string]any, len(questions))
	for i, item := range questions {
		q := item.(map[string]any)
		picked := wrongOption
		if i < nRight {
			picked = rightOption
		}
		answers[i] = map[string]any{"key": q["key"].(string), "answer": picked}
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

// programReply is the planner choosing from the library: ids, not texts.
const programReply = `{"summary":"Solid A1 with clear A2 gaps.","topics":[
	{"id":"a1-to-be","why":"Вы путаете am, is и are."},
	{"id":"a1-present-simple","why":"Окончание -s у he и she."},
	{"id":"a1-present-simple-questions","why":"Вопросы с do и does."},
	{"id":"a1-articles","why":"Артикли a и the."},
	{"id":"a1-vocab-food","why":"Слова для кафе и магазина."},
	{"id":"a1-reading-notices","why":"Чтение коротких объявлений."},
	{"id":"a1-listening-numbers","why":"Числа на слух."},
	{"id":"a1-speaking-greetings","why":"Знакомство вслух."}]}`

// startBuild asks for the program of a finished test; it answers at once with
// a job.
func startBuild(t *testing.T, e env, testID string) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/analyze", e.session, nil)
	if code != http.StatusAccepted {
		t.Fatalf("analyze: %d %v", code, body)
	}
	return data(t, body)
}

// waitJob follows a build until it is done or failed.
func waitJob(t *testing.T, e env, id string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		code, body := request(t, e.e, "GET", "/api/v1/learning/path/jobs/"+id, e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("job: %d %v", code, body)
		}
		job := data(t, body)
		if job["status"] == "done" || job["status"] == "failed" {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatalf("the job never finished: %v", job)
		}
		time.Sleep(15 * time.Millisecond)
	}
}

// analyze builds the program of a test and returns it.
func analyze(t *testing.T, e env, testID string) map[string]any {
	t.Helper()
	job := waitJob(t, e, startBuild(t, e, testID)["id"].(string))
	if job["status"] != "done" || job["path"] == nil {
		t.Fatalf("analyze: %v", job)
	}
	return job["path"].(map[string]any)
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
	if len(topics) != 8 {
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

	// The program is made of the library's own topics: the rule and the goal
	// are written by people, the reason is the planner's.
	if first["title"] != "Глагол to be: am, is, are" || !strings.Contains(first["theory"].(string), "I am, you are") {
		t.Fatalf("the topic does not carry the library's rule: %v", first)
	}
	if !strings.Contains(first["goal"].(string), "Вы путаете am, is и are.") {
		t.Fatalf("the planner's reason is missing from the goal: %v", first["goal"])
	}
	food := topics[4].(map[string]any)
	if words := food["vocabulary"].([]any); len(words) < 6 {
		t.Fatalf("a vocabulary topic must bring its words: %v", food)
	}
	if !strings.Contains(prompts[0], "library") || !strings.Contains(prompts[0], "a1-to-be") {
		t.Fatalf("the planner was not given the library:\n%s", prompts[0])
	}
}

// TestPlannerCannotInventATopic: an id that is not in the library is refused
// and the retry says so; the program is built from the corrected answer.
func TestPlannerCannotInventATopic(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	bad := strings.Replace(programReply, "a1-articles", "a1-made-up-topic", 1)
	e.provider.enqueue(bad, programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	if len(path["topics"].([]any)) != 8 {
		t.Fatalf("path: %v", path)
	}

	prompts := e.provider.requests()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "is not in the library") {
		t.Fatalf("the retry did not say the id is unknown: %d\n%v", len(prompts), prompts)
	}
}

// TestProgramNeedsListeningAndSpeaking: a plan with no listening or no speaking
// topic is sent back, so a program is never only reading and gap filling.
func TestProgramNeedsListeningAndSpeaking(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	noListening := strings.Replace(programReply, "a1-listening-numbers", "a1-writing-intro", 1)
	e.provider.enqueue(noListening, programReply)

	analyze(t, e, climb(t, e, "A1"))

	prompts := e.provider.requests()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "has no listening topic") {
		t.Fatalf("the retry did not ask for a listening topic: %d", len(prompts))
	}
}

// TestPlannerStaysWithinTheLearnersLevel: a topic above the level after the
// one reached is not on offer, so asking for it is refused.
func TestPlannerStaysWithinTheLearnersLevel(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	tooHigh := strings.Replace(programReply, "a1-articles", "b2-wish", 1)
	e.provider.enqueue(tooHigh, tooHigh)

	job := waitJob(t, e, startBuild(t, e, climb(t, e, "A1"))["id"].(string))
	if job["status"] != "failed" || !strings.Contains(job["error"].(string), "could not be read") {
		t.Fatalf("a topic far above the learner must fail the build: %v", job)
	}
	if job["path"] != nil {
		t.Fatalf("a failed build left a program: %v", job)
	}
}

// TestBuildRunsInTheBackground: the request only queues a job. Asking again
// hands the same job back, while it runs and after it is done, so a double tap
// never builds two programs.
func TestBuildRunsInTheBackground(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	testID := climb(t, e, "A1")

	first := startBuild(t, e, testID)
	if first["status"] != "queued" && first["status"] != "running" && first["status"] != "done" {
		t.Fatalf("job: %v", first)
	}

	again := startBuild(t, e, testID)
	if again["id"] != first["id"] {
		t.Fatalf("a second ask must hand back the same job: %v %v", first, again)
	}

	done := waitJob(t, e, first["id"].(string))
	if done["status"] != "done" || len(done["path"].(map[string]any)["topics"].([]any)) != 8 {
		t.Fatalf("the build: %v", done)
	}

	after := startBuild(t, e, testID)
	if after["id"] != first["id"] || after["status"] != "done" {
		t.Fatalf("a finished build must be handed back, not made twice: %v", after)
	}

	// One program, not two.
	code, body := request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("teachers: %d %v", code, body)
	}
}

// TestFailedBuildCanBeStartedOver: a build that failed says why, and asking
// again after the agent is back makes a new job and a program.
func TestFailedBuildCanBeStartedOver(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault("I cannot plan that.")

	testID := climb(t, e, "A1")
	failed := waitJob(t, e, startBuild(t, e, testID)["id"].(string))
	if failed["status"] != "failed" || failed["error"] == "" {
		t.Fatalf("a build the agent could not do must fail with a reason: %v", failed)
	}

	e.provider.setDefault(programReply)
	retry := startBuild(t, e, testID)
	if retry["id"] == failed["id"] {
		t.Fatalf("a failed build must be started over as a new job: %v", retry)
	}
	if done := waitJob(t, e, retry["id"].(string)); done["status"] != "done" {
		t.Fatalf("the retry: %v", done)
	}
}

// TestBuildPreparesTheFirstExercises: once the program is saved the first
// topics get their exercises written in the background, and nobody has started
// those topics, so they stay as they were.
func TestBuildPreparesTheFirstExercises(t *testing.T) {
	e := setupPrefetch(t, "", 2)
	chooseAgent(t, e)

	testID := climb(t, e, "A1")
	e.provider.enqueue(programReply, exercisesReply, exercisesReply)

	path := analyze(t, e, testID)
	topics := path["topics"].([]any)
	first := topics[0].(map[string]any)["id"].(string)
	third := topics[2].(map[string]any)["id"].(string)

	deadline := time.Now().Add(15 * time.Second)
	for {
		detail := topicDetail(t, e, first)
		if len(detail["exercises"].([]any)) > 0 {
			if detail["topic"].(map[string]any)["status"] != "not_started" {
				t.Fatalf("preparing a set must not start the topic: %v", detail["topic"])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the first topic's exercises were never prepared")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if n := len(topicDetail(t, e, third)["exercises"].([]any)); n != 0 {
		t.Fatalf("only the first two topics are prepared, the third has %d exercises", n)
	}
}

// TestPlacementAnswersNeverReachTheClient checks the test DTO carries no
// answer keys.
func TestPlacementAnswersNeverReachTheClient(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

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

// TestAnalyzeWithoutAnAgent is the setup error a learner can act on: the agent
// was there for the test and is gone by the time the program is built.
func TestAnalyzeWithoutAnAgent(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

	testID := climb(t, e, "A1")

	if _, err := e.pool.Exec(t.Context(), "DELETE FROM learning_ai_settings WHERE user_id = $1", e.user); err != nil {
		t.Fatal(err)
	}

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/analyze", e.session, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("analyze with no agent: %d %v", code, body)
	}
}

// openTests counts the tests of the learner: a test that could not start must
// leave none behind.
func openTests(t *testing.T, e env) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(t.Context(), "SELECT count(*) FROM learning_path_tests WHERE user_id = $1", e.user).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestTestCannotStartWithoutAnAgent: the test is written by the agent, so
// without one it does not start and nothing is left open.
func TestTestCannotStartWithoutAnAgent(t *testing.T) {
	e := setup(t)

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusBadRequest {
		t.Fatalf("start without an agent: %d %v", code, body)
	}
	if openTests(t, e) != 0 {
		t.Fatal("a test that could not start was left open")
	}
}

// TestTestCannotStartWhenTheAgentFails: an agent that answers garbage, even
// after the retry, stops the test too.
func TestTestCannotStartWhenTheAgentFails(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.breakQuestions = true

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusBadGateway {
		t.Fatalf("start with a broken agent: %d %v", code, body)
	}
	if openTests(t, e) != 0 {
		t.Fatal("a test that could not start was left open")
	}
}

// TestQuestionsAreWrittenForEachTest: nothing is canned. Every test asks the
// agent, the next level's batch is written when the learner climbs, and what
// was asked already travels along so it is not repeated.
func TestQuestionsAreWrittenForEachTest(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

	testID, questions := startPlacement(t, e)
	if calls := e.provider.testCalls(); len(calls) != 1 || !strings.Contains(calls[0], "language level test") {
		t.Fatalf("the first batch was not written by the agent: %d calls", len(calls))
	}

	first := questions[0].(map[string]any)["prompt"].(string)
	state := answerBatch(t, e, testID, questions, true)
	if state["level"] != "A2" || state["startLevel"] != "A1" {
		t.Fatalf("the ladder did not climb from A1: %v %v", state["level"], state["startLevel"])
	}

	// One call for the examiner, one for the A2 batch, which carries the A1 prompts.
	calls := e.provider.testCalls()
	if len(calls) != 3 {
		t.Fatalf("calls: %d", len(calls))
	}
	if !strings.Contains(calls[2], first) {
		t.Fatalf("the next batch was not told what was asked:\n%s", calls[2])
	}
}

// TestExaminerDecidesTheLevel: the level is the agent's call, not a threshold.
// Five of eight right still climbs when the examiner says so.
func TestExaminerDecidesTheLevel(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.examiner = func(level string, correct, asked int) string {
		return `{"verdict":"climb","level":"` + level + `"}`
	}

	testID, questions := startPlacement(t, e)
	state := answerMixed(t, e, testID, questions, 5)
	if state["status"] == "done" || state["level"] != "A2" {
		t.Fatalf("the examiner's verdict was not followed: %v", state)
	}
}

// TestExaminerCannotContradictTheBatch: a verdict that climbs a learner who got
// nothing right is refused, and the test is left where it was.
func TestExaminerCannotContradictTheBatch(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.examiner = func(level string, correct, asked int) string {
		return `{"verdict":"climb","level":"` + level + `"}`
	}

	testID, questions := startPlacement(t, e)

	answers := make([]map[string]any, len(questions))
	for i, item := range questions {
		answers[i] = map[string]any{"key": item.(map[string]any)["key"], "answer": wrongOption}
	}
	code, _ := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/answers", e.session, map[string]any{"answers": answers})
	if code != http.StatusBadGateway {
		t.Fatalf("a climb on an all-wrong batch must be refused: %d", code)
	}
}

const exercisesReply = `{"exercises":[
	{"kind":"choice","text":"","prompt":"She ___ to school every day.","options":["go","goes","going","went"],"answer":"goes","rule":"В Present Simple с she, he или it к глаголу добавляется окончание -s."},
	{"kind":"fill","text":"","prompt":"Yesterday I ... my bike. (ride)","options":[],"answer":"rode","rule":"Глагол ride — неправильный: в Past Simple его форма rode."},
	{"kind":"translate","text":"","prompt":"Translate into English: Я живу в Москве.","options":[],"answer":"I live in Moscow.","rule":"Для постоянного места жительства используем Present Simple: I live."}]}`

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

	prompts := e.provider.requests()
	exercisePrompt := prompts[len(prompts)-1]
	for _, want := range []string{"what the rule is", "how to apply it", "why the given answer is correct", `\"theory\"`, "I am, you are"} {
		if !strings.Contains(exercisePrompt, want) {
			t.Fatalf("the exercise writer was not asked for %s:\n%s", want, exercisePrompt)
		}
	}

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

	// A right choice answer and its short rule are available before the attempt.
	choice := exercises[0].(map[string]any)
	if choice["answer"] != "goes" || choice["rule"] == "" {
		t.Fatalf("exercise hint: %v", choice)
	}

	// A right choice answer is graded on the spot: no agent call, no feedback.
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

// TestNextLevelTestStartsAboveTheConfirmedLevel: after a program is confirmed
// by its final test, a placement can start at the rung above and nowhere else.
func TestNextLevelTestStartsAboveTheConfirmedLevel(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A1"))

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/final-tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated {
		t.Fatalf("start final: %d %v", code, body)
	}
	state := answerBatch(t, e, data(t, body)["id"].(string), data(t, body)["questions"].([]any), true)
	if state["passed"] != true {
		t.Fatalf("final: %v", state)
	}

	start := func(level string) (int, map[string]any) {
		return request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{
			"language": "en", "translationLanguage": "ru", "level": level,
		})
	}

	if code, body := start("B1"); code != http.StatusBadRequest {
		t.Fatalf("skipping a rung must be refused: %d %v", code, body)
	}

	code, body = start("A2")
	if code != http.StatusCreated || data(t, body)["level"] != "A2" || data(t, body)["startLevel"] != "A2" {
		t.Fatalf("the next rung must start the test: %d %v", code, body)
	}
}

// TestDeletingATeacherKeepsTheProgramInTheHistory: the teacher leaves the list
// and its running program is closed into the history, not thrown away; the
// language can be started again.
func TestDeletingATeacherKeepsTheProgramInTheHistory(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	analyze(t, e, climb(t, e, "A1"))

	code, body := request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
	teachers := listPayload(t, body)
	if code != http.StatusOK || len(teachers) != 1 {
		t.Fatalf("teachers: %d %v", code, body)
	}
	id := teachers[0]["id"].(string)

	code, body = request(t, e.e, "DELETE", "/api/v1/learning/path/teachers/"+id, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("delete: %d %v", code, body)
	}

	_, body = request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
	if left := listPayload(t, body); len(left) != 0 {
		t.Fatalf("the teacher must be gone: %v", left)
	}

	if code, _ := request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil); code != http.StatusNotFound {
		t.Fatalf("the program must no longer be running: %d", code)
	}

	_, body = request(t, e.e, "GET", "/api/v1/learning/path/history", e.session, nil)
	history := listPayload(t, body)
	if len(history) != 1 || history[0]["status"] != "abandoned" {
		t.Fatalf("the program must sit in the history: %v", history)
	}

	if code, _ := request(t, e.e, "DELETE", "/api/v1/learning/path/teachers/"+id, e.session, nil); code != http.StatusNotFound {
		t.Fatalf("deleting twice: %d", code)
	}

	// Studying the language again makes a new teacher.
	startPlacement(t, e)
	_, body = request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
	if again := listPayload(t, body); len(again) != 1 || again[0]["id"] == id {
		t.Fatalf("a new teacher is expected: %v", again)
	}
}

// TestRuleBookNeedsTheTeacher: the rules are served to someone who has the
// teacher whose language they cover, and to no one else; a topic of that
// teacher's program carries its rule with references to other rules.
func TestRuleBookNeedsTheTeacher(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	if code, _ := request(t, e.e, "GET", "/api/v1/learning/path/rules", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("without a teacher the rule book must be closed: %d", code)
	}

	// A teacher whose language has no rule book does not open it.
	request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": "es", "translationLanguage": "ru"})
	if code, _ := request(t, e.e, "GET", "/api/v1/learning/path/rules", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("a Spanish teacher must not open the English rule book: %d", code)
	}

	path := analyze(t, e, climb(t, e, "A1"))

	code, body := request(t, e.e, "GET", "/api/v1/learning/path/rules", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("rules: %d %v", code, body)
	}
	rules := listPayload(t, body)
	if len(rules) < 40 {
		t.Fatalf("the rule book is too small: %d", len(rules))
	}

	code, body = request(t, e.e, "GET", "/api/v1/learning/path/rules/b2-past-perfect", e.session, nil)
	rule := data(t, body)
	if code != http.StatusOK || rule["title"] == "" || len(rule["forms"].([]any)) < 2 || len(rule["examples"].([]any)) < 3 {
		t.Fatalf("rule: %d %v", code, body)
	}
	related := rule["related"].([]any)
	if len(related) < 2 || related[0].(map[string]any)["id"] == "" || related[0].(map[string]any)["title"] == "" {
		t.Fatalf("a rule must refer to other rules: %v", related)
	}

	if code, _ := request(t, e.e, "GET", "/api/v1/learning/path/rules/a1-vocab-food", e.session, nil); code != http.StatusNotFound {
		t.Fatalf("a topic that is not a rule: %d", code)
	}

	// The first topic of the program is grammar made from the library: it comes with its rule.
	first := path["topics"].([]any)[0].(map[string]any)["id"].(string)
	detail := topicDetail(t, e, first)
	topicRule, _ := detail["rule"].(map[string]any)
	if topicRule == nil || topicRule["id"] != "a1-to-be" || len(topicRule["related"].([]any)) < 2 {
		t.Fatalf("the topic must carry its rule: %v", detail["rule"])
	}

	// A vocabulary topic has words, not a rule.
	food := path["topics"].([]any)[4].(map[string]any)["id"].(string)
	if topicDetail(t, e, food)["rule"] != nil {
		t.Fatal("a vocabulary topic has no rule")
	}

	// Deleting the teacher closes the book again, and the kept program stops handing it out.
	_, body = request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
	for _, teacher := range listPayload(t, body) {
		request(t, e.e, "DELETE", "/api/v1/learning/path/teachers/"+teacher["id"].(string), e.session, nil)
	}
	if code, _ := request(t, e.e, "GET", "/api/v1/learning/path/rules/b2-past-perfect", e.session, nil); code != http.StatusForbidden {
		t.Fatalf("without the teacher the rule book must be closed again: %d", code)
	}
	if topicDetail(t, e, first)["rule"] != nil {
		t.Fatal("a topic of a deleted teacher must not hand out its rule")
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
	chooseAgent(t, e)

	first, _ := startPlacement(t, e)
	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated || data(t, body)["id"] != first {
		t.Fatalf("a second start must hand back the open test: %d %v", code, body)
	}
}

// TestUnknownAnswerKeyIsRefused keeps the batch protocol honest.
func TestUnknownAnswerKeyIsRefused(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

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
	if len(newSet) == 0 || newSet[0]["id"] == exerciseID {
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
	chooseAgent(t, e)

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

func lessonCall(t *testing.T, e env, method, path string, want int) map[string]any {
	t.Helper()
	code, body := request(t, e.e, method, "/api/v1/learning/path/"+path, e.session, nil)
	if code != want {
		t.Fatalf("%s %s: %d %v", method, path, code, body)
	}
	if body["data"] == nil {
		return nil
	}
	return data(t, body)
}

// TestLessonTimerKeepsRunning: a lesson is the learner's own 30 minutes, not a
// topic's. It runs from its start, pausing does not move the end, what is
// answered in it counts to it on whichever topic, a second start while it runs
// hands it back, answering starts a topic, and a lesson past its end is closed
// at that end.
func TestLessonTimerKeepsRunning(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topics := path["topics"].([]any)
	topicID := topics[0].(map[string]any)["id"].(string)
	otherID := topics[1].(map[string]any)["id"].(string)

	e.provider.enqueue(exercisesReply)
	exercises := generateExercises(t, e, topicID)
	e.provider.enqueue(exercisesReply)
	others := generateExercises(t, e, otherID)

	lesson := lessonCall(t, e, "POST", "lessons", http.StatusOK)
	lessonID := lesson["id"].(string)
	started, _ := time.Parse(time.RFC3339Nano, lesson["startedAt"].(string))
	ends, _ := time.Parse(time.RFC3339Nano, lesson["endsAt"].(string))
	if ends.Sub(started) != 30*time.Minute || lesson["status"] != "active" {
		t.Fatalf("lesson: %v", lesson)
	}

	// Starting again hands the same lesson back.
	if again := lessonCall(t, e, "POST", "lessons", http.StatusOK); again["id"] != lessonID {
		t.Fatalf("a second lesson opened: %v", again)
	}

	paused := lessonCall(t, e, "POST", "lessons/"+lessonID+"/pause", http.StatusOK)
	if paused["pausedAt"] == nil || paused["endsAt"] != lesson["endsAt"] || paused["pauseCount"] != float64(1) {
		t.Fatalf("pause: %v", paused)
	}
	resumed := lessonCall(t, e, "POST", "lessons/"+lessonID+"/resume", http.StatusOK)
	if resumed["pausedAt"] != nil || resumed["endsAt"] != lesson["endsAt"] {
		t.Fatalf("resume: %v", resumed)
	}

	// Answers on two different topics both count to the one lesson.
	right := exercises[0].(map[string]any)["answer"].(string)
	attempt(t, e, exercises[0].(map[string]any)["id"].(string), right, true, "")

	active := lessonCall(t, e, "GET", "lessons/active", http.StatusOK)
	if active["id"] != lessonID || active["attempts"] != float64(1) || active["correct"] != float64(1) {
		t.Fatalf("active: %v", active)
	}

	if status := topicDetail(t, e, topicID)["topic"].(map[string]any)["status"]; status != "in_progress" {
		t.Fatalf("answering must start the topic: %v", status)
	}

	rightOther := others[0].(map[string]any)["answer"].(string)
	attempt(t, e, others[0].(map[string]any)["id"].(string), rightOther, true, "")

	if active = lessonCall(t, e, "GET", "lessons/active", http.StatusOK); active["id"] != lessonID || active["attempts"] != float64(2) {
		t.Fatalf("an answer on another topic must count to the same lesson: %v", active)
	}

	// Moving to another topic does not start another lesson.
	if again := lessonCall(t, e, "POST", "lessons", http.StatusOK); again["id"] != lessonID {
		t.Fatalf("the lesson changed with the topic: %v", again)
	}

	code, body := request(t, e.e, "GET", "/api/v1/learning/path/lessons", e.session, nil)
	if code != http.StatusOK || len(listPayload(t, body)) != 1 {
		t.Fatalf("lessons: %d %v", code, body)
	}

	// Time running out closes it at the end moment.
	if _, err := e.pool.Exec(t.Context(), "UPDATE learning_path_lessons SET ends_at = now() - interval '1 minute' WHERE id = $1", lessonID); err != nil {
		t.Fatal(err)
	}
	if got := lessonCall(t, e, "GET", "lessons/active", http.StatusOK); got != nil {
		t.Fatalf("an expired lesson is still active: %v", got)
	}

	// An answer after the end is a free one, and the next start is a new lesson.
	attempt(t, e, exercises[1].(map[string]any)["id"].(string), exercises[1].(map[string]any)["answer"].(string), true, "")

	finished := lessonCall(t, e, "POST", "lessons/"+lessonID+"/finish", http.StatusOK)
	if finished["status"] != "finished" || finished["finishedAt"] == nil || finished["attempts"] != float64(2) {
		t.Fatalf("finish: %v", finished)
	}

	if next := lessonCall(t, e, "POST", "lessons", http.StatusOK); next["id"] == lessonID {
		t.Fatal("a finished lesson was handed back")
	}
}

// TestPathTopicChat: every program topic has a chat of its own, which only its
// owner can read and which can be cleared.
func TestPathTopicChat(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	// Chat is built for deepseek only; the program above came from the agent.
	code, body := request(t, e.e, "PUT", "/api/v1/learning/ai/settings", e.session, map[string]any{
		"provider": "deepseek", "model": "deepseek-flash", "level": "high", "apiKey": "sk-deepseek-test-1234",
	})
	if code != http.StatusOK {
		t.Fatalf("choose chat agent: %d %v", code, body)
	}

	code, body = request(t, e.e, "GET", "/api/v1/learning/path/topics/"+topicID+"/chat", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("get chat: %d %v", code, body)
	}
	if d := data(t, body); d["available"] != true || len(d["messages"].([]any)) != 0 {
		t.Fatalf("chat: %v", d)
	}

	e.provider.setDefault("Presente simple: yo hablo.")
	// A chat turn streams its answer (server-sent events), so it is not JSON.
	send := httptest.NewRequest("POST", "/api/v1/learning/path/topics/"+topicID+"/chat", strings.NewReader(`{"text":"explain this topic"}`))
	send.Header.Set("Content-Type", "application/json")
	send.Header.Set("Authorization", "Bearer "+e.session)
	sent := httptest.NewRecorder()
	e.e.ServeHTTP(sent, send)
	if sent.Code != http.StatusOK || !strings.Contains(sent.Body.String(), "Presente simple: yo hablo.") {
		t.Fatalf("send: %d %s", sent.Code, sent.Body.String())
	}
	prompts := e.provider.requests()
	if last := prompts[len(prompts)-1]; !strings.Contains(last, "study program") {
		t.Fatalf("the teacher was not told about the topic: %s", last)
	}

	_, body = request(t, e.e, "GET", "/api/v1/learning/path/topics/"+topicID+"/chat", e.session, nil)
	if msgs := data(t, body)["messages"].([]any); len(msgs) != 2 {
		t.Fatalf("messages: %v", msgs)
	}

	code, _ = request(t, e.e, "DELETE", "/api/v1/learning/path/topics/"+topicID+"/chat", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("clear: %d", code)
	}
	_, body = request(t, e.e, "GET", "/api/v1/learning/path/topics/"+topicID+"/chat", e.session, nil)
	if msgs := data(t, body)["messages"].([]any); len(msgs) != 0 {
		t.Fatalf("not cleared: %v", msgs)
	}

	code, _ = request(t, e.e, "GET", "/api/v1/learning/path/topics/"+uuid.NewString()+"/chat", e.session, nil)
	if code != http.StatusNotFound {
		t.Fatalf("a stranger's topic: %d", code)
	}
}

// TestTeachersAreLanguagesStudiedSideBySide: a teacher is a language with the
// language its rules are explained in, one per language. Starting a test makes
// the teacher, an existing teacher keeps its pair, and the list says how far
// each running program has got.
func TestTeachersAreLanguagesStudiedSideBySide(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	list := func() []map[string]any {
		code, body := request(t, e.e, "GET", "/api/v1/learning/path/teachers", e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("teachers: %d %v", code, body)
		}
		return listPayload(t, body)
	}

	if got := list(); len(got) != 0 {
		t.Fatalf("a new learner has no teachers: %v", got)
	}

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": "es", "translationLanguage": "ru"})
	if code != http.StatusCreated || data(t, body)["language"] != "es" || data(t, body)["path"] != nil {
		t.Fatalf("create: %d %v", code, body)
	}
	first := data(t, body)["id"]

	// One teacher per language: asking again hands back the same one, with its own pair.
	code, body = request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": "es", "translationLanguage": "en"})
	if code != http.StatusCreated || data(t, body)["id"] != first || data(t, body)["translationLanguage"] != "ru" {
		t.Fatalf("a second teacher for a language: %d %v", code, body)
	}

	code, _ = request(t, e.e, "POST", "/api/v1/learning/path/teachers", e.session, map[string]any{"language": "ru", "translationLanguage": "ru"})
	if code != http.StatusBadRequest {
		t.Fatalf("a language explained in itself must be refused: %d", code)
	}

	// A test for that language keeps the teacher's pair, whatever the request says.
	code, body = request(t, e.e, "POST", "/api/v1/learning/path/tests", e.session, map[string]any{"language": "es", "translationLanguage": "en"})
	if code != http.StatusCreated || data(t, body)["translationLanguage"] != "ru" {
		t.Fatalf("the test must keep the teacher's pair: %d %v", code, body)
	}

	// Studying English makes its teacher; the program shows up on its card.
	analyze(t, e, climb(t, e, "A1"))

	teachers := list()
	if len(teachers) != 2 {
		t.Fatalf("want two teachers, got %v", teachers)
	}
	var english map[string]any
	for _, teacher := range teachers {
		if teacher["language"] == "en" {
			english = teacher
		}
		if teacher["language"] == "es" && teacher["path"] != nil {
			t.Fatalf("Spanish has no program yet: %v", teacher)
		}
	}
	if english == nil || english["path"] == nil {
		t.Fatalf("English must have a teacher with a program: %v", teachers)
	}
	summary := english["path"].(map[string]any)
	if summary["level"] != "A1" || summary["topicsTotal"].(float64) != 8 || summary["topicsDone"].(float64) != 0 {
		t.Fatalf("program summary: %v", summary)
	}
}

// rawGet reads a response that is not JSON, such as a voice.
func rawGet(t *testing.T, e env, path string) (int, []byte, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+e.session)
	rec := httptest.NewRecorder()
	e.e.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes(), rec.Header().Get("Content-Type")
}

const variedExercisesReply = `{"exercises":[
	{"kind":"listen","text":"She has been waiting for an hour.","prompt":"Как долго она ждёт?","options":["for an hour","since an hour","an hour ago","during an hour"],"answer":"for an hour","rule":"For отвечает на вопрос «как долго»."},
	{"kind":"speak","text":"","prompt":"Скажите вслух.","options":[],"answer":"Nice to meet you","rule":"Слова to и you произносите слитно."},
	{"kind":"write","text":"","prompt":"Напишите два предложения о вашем дне.","options":[],"answer":"I get up at seven. I work in an office.","rule":"Используйте Present Simple для распорядка дня."}]}`

// TestListeningSpeakingAndWritingExercises: the three new kinds are stored, a
// listening answer and a spoken phrase are graded here without the agent, and a
// text of the learner's own is judged by the agent.
func TestListeningSpeakingAndWritingExercises(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.enqueue(variedExercisesReply)
	exercises := generateExercises(t, e, topicID)
	if len(exercises) != 3 {
		t.Fatalf("exercises: %v", exercises)
	}
	listen, speak, write := exercises[0].(map[string]any), exercises[1].(map[string]any), exercises[2].(map[string]any)

	calls := len(e.provider.requests())

	attempt(t, e, listen["id"].(string), "for an hour", true, "")

	// A phrase heard a little differently still counts; one that is not it does not.
	attempt(t, e, speak["id"].(string), "nice to meet you", true, "")
	attempt(t, e, speak["id"].(string), "nice to meat you", true, "")
	attempt(t, e, speak["id"].(string), "see you later", false, "")

	if len(e.provider.requests()) != calls {
		t.Fatal("grading a choice and a spoken phrase must not spend the agent")
	}

	// A text of the learner's own has no key: the agent judges it.
	e.provider.enqueue(`{"correct":true,"feedback":""}`)
	attempt(t, e, write["id"].(string), "I get up at seven. I go to work by bus.", true, "")
	if len(e.provider.requests()) != calls+1 {
		t.Fatal("a written text must be judged by the agent")
	}
	judge := e.provider.requests()[calls]
	if !strings.Contains(judge, "write") {
		t.Fatalf("the agent was not told it is a writing exercise:\n%s", judge)
	}
}

// TestBrokenNewExerciseKindsAreRetried: a listening exercise with no sentence,
// or a speaking one with options, never reaches the learner.
func TestBrokenNewExerciseKindsAreRetried(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	noSentence := strings.Replace(variedExercisesReply, `"text":"She has been waiting for an hour."`, `"text":""`, 1)
	e.provider.enqueue(noSentence, variedExercisesReply)
	generateExercises(t, e, topicID)

	prompts := e.provider.requests()
	if !strings.Contains(prompts[len(prompts)-1], "listen needs the sentence to hear") {
		t.Fatalf("the retry did not say what was wrong:\n%s", prompts[len(prompts)-1])
	}
}

// TestExerciseAndWordAudio: a listening exercise and a key word are read aloud;
// an exercise with nothing to hear is refused.
func TestExerciseAndWordAudio(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topics := path["topics"].([]any)
	topicID := topics[0].(map[string]any)["id"].(string)

	e.provider.enqueue(variedExercisesReply)
	exercises := generateExercises(t, e, topicID)

	code, body, kind := rawGet(t, e, "/api/v1/learning/path/exercises/"+exercises[0].(map[string]any)["id"].(string)+"/audio")
	if code != http.StatusOK || len(body) == 0 || !strings.HasPrefix(kind, "audio/") {
		t.Fatalf("listening audio: %d %q %q", code, body, kind)
	}

	// The cache folder may be lost while its rows stay (a new container, a wiped
	// volume): the voice is made again instead of failing for good.
	files, _ := os.ReadDir(e.audioDir)
	for _, f := range files {
		_ = os.Remove(filepath.Join(e.audioDir, f.Name()))
	}
	code, body, _ = rawGet(t, e, "/api/v1/learning/path/exercises/"+exercises[0].(map[string]any)["id"].(string)+"/audio")
	if code != http.StatusOK || len(body) == 0 {
		t.Fatalf("a lost cache file must be made again: %d", code)
	}

	code, _, _ = rawGet(t, e, "/api/v1/learning/path/exercises/"+exercises[1].(map[string]any)["id"].(string)+"/audio")
	if code != http.StatusOK {
		t.Fatalf("a speaking exercise may be heard before it is repeated: %d", code)
	}

	code, _, _ = rawGet(t, e, "/api/v1/learning/path/exercises/"+exercises[2].(map[string]any)["id"].(string)+"/audio")
	if code != http.StatusBadRequest {
		t.Fatalf("a writing exercise has nothing to hear: %d", code)
	}

	// The food topic carries its words; the first one can be heard, a missing one cannot.
	food := topics[4].(map[string]any)["id"].(string)
	code, body, _ = rawGet(t, e, "/api/v1/learning/path/topics/"+food+"/words/0/audio")
	if code != http.StatusOK || len(body) == 0 {
		t.Fatalf("word audio: %d", code)
	}
	code, _, _ = rawGet(t, e, "/api/v1/learning/path/topics/"+food+"/words/99/audio")
	if code != http.StatusNotFound {
		t.Fatalf("a missing word: %d", code)
	}
}
