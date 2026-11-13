package learning_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/ratelimit"
	"learning_app/backend/internal/secretbox"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

// deepseek stands in for the provider: it streams a fixed answer for chat turns
// and writes a fixed summary for the cheap non-streaming call, and remembers
// every request body.
type deepseek struct {
	mu           sync.Mutex
	status       int
	promptTokens int
	turns        []map[string]any
	summaries    []map[string]any
}

func (d *deepseek) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.URL.Path != "/chat/completions" {
		_, _ = w.Write([]byte(`{"data":[]}`))

		return
	}

	if d.status != http.StatusOK {
		w.WriteHeader(d.status)

		return
	}

	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	if body["stream"] != true {
		d.summaries = append(d.summaries, body)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"SUMMARY: the learner prefers short answers"}}]}`))

		return
	}

	d.turns = append(d.turns, body)

	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Answer \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"%d.\"}}]}\n\n", len(d.turns))
	_, _ = fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":5,\"prompt_cache_hit_tokens\":0}}\n\ndata: [DONE]\n\n", d.promptTokens)
}

type chatEnv struct {
	e        *echo.Echo
	q        *db.Queries
	pool     *pgxpool.Pool
	provider *deepseek
	user     uuid.UUID
	session  string
	topic    string
	entry    string
}

// looseRate keeps the per-user rate limit out of the way of tests that are not
// about it.
var looseRate = ratelimit.Config{Burst: 1000, Every: time.Millisecond, InFlight: 100}

func setupChat(t *testing.T) chatEnv { return setupChatRate(t, looseRate) }

func setupChatRate(t *testing.T, rate ratelimit.Config) chatEnv {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)

	provider := &deepseek{status: http.StatusOK, promptTokens: 300}
	srv := httptest.NewServer(provider)
	t.Cleanup(srv.Close)

	cfg := config.Config{AuthSecret: authSecret, ChatRate: rate, LLMURLs: llm.URLs{Anthropic: srv.URL, OpenAI: srv.URL, DeepSeek: srv.URL, Kimi: srv.URL}}
	e := server.New(cfg, pool, testdb.AccountPool(t))
	session := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	topicID := topic(t, e, session)

	code, body := request(t, e, "POST", "/api/v1/learning/entries", session, map[string]any{"topicId": topicID, "kind": "phrase", "text": "break a leg", "status": "learning"}, "")
	if code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	return chatEnv{e: e, q: q, pool: pool, provider: provider, user: user, session: session, topic: topicID, entry: data(t, body)["id"].(string)}
}

func (c chatEnv) path() string { return "/api/v1/learning/entries/" + c.entry + "/chat" }

const defaultKey = "sk-deepseek-1234"

func (c chatEnv) useDeepSeek(t *testing.T) { c.useKey(t, defaultKey) }

// useKey chooses DeepSeek with the given API key.
func (c chatEnv) useKey(t *testing.T, key string) {
	t.Helper()

	if code, body := request(t, c.e, "PUT", settingsPath, c.session, choice("deepseek", "deepseek-v4-pro", "high", key), ""); code != 200 {
		t.Fatalf("settings: %d %v", code, body)
	}
}

// setUsage writes a day's counter as if that many tokens had been spent.
func (c chatEnv) setUsage(t *testing.T, scope, subject string, daysAgo int, tokens int64) {
	t.Helper()

	_, err := c.pool.Exec(context.Background(), `INSERT INTO learning_chat_usage (scope, subject, day, tokens, turns)
		VALUES ($1, $2, (now() AT TIME ZONE 'utc')::date - $3::int, $4, 1)
		ON CONFLICT (scope, subject, day) DO UPDATE SET tokens = EXCLUDED.tokens`, scope, subject, daysAgo, tokens)
	if err != nil {
		t.Fatal(err)
	}
}

type sseEvent struct {
	name string
	data map[string]any
}

// post sends a question as it is and returns the raw response.
func (c chatEnv) post(t *testing.T, text string) *httptest.ResponseRecorder {
	t.Helper()

	return c.postTo(t, c.path(), text)
}

func (c chatEnv) postTo(t *testing.T, path, text string) *httptest.ResponseRecorder {
	t.Helper()

	raw, _ := json.Marshal(map[string]any{"text": text})
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.session)

	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)

	return rec
}

// ask posts a question and returns the status with the parsed event stream (or,
// for a refusal, the JSON envelope as one event named "body").
func (c chatEnv) ask(t *testing.T, text string) (int, []sseEvent) {
	t.Helper()

	return c.askAt(t, c.path(), text)
}

func (c chatEnv) askAt(t *testing.T, path, text string) (int, []sseEvent) {
	t.Helper()

	rec := c.postTo(t, path, text)

	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)

		return rec.Code, []sseEvent{{name: "body", data: body}}
	}

	var out []sseEvent

	for _, block := range strings.Split(strings.TrimSpace(rec.Body.String()), "\n\n") {
		lines := strings.SplitN(block, "\n", 2)
		ev := sseEvent{name: strings.TrimPrefix(lines[0], "event: ")}
		_ = json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &ev.data)
		out = append(out, ev)
	}

	return rec.Code, out
}

func (c chatEnv) history(t *testing.T) (bool, []any) {
	t.Helper()

	code, body := request(t, c.e, "GET", c.path(), c.session, nil, "")
	if code != 200 {
		t.Fatalf("get chat: %d %v", code, body)
	}

	d := data(t, body)

	return d["available"].(bool), d["messages"].([]any)
}

func messagesOf(turn map[string]any) []map[string]any {
	var out []map[string]any

	for _, m := range turn["messages"].([]any) {
		out = append(out, m.(map[string]any))
	}

	return out
}

func TestChatNeedsAnAgentButStaysReadable(t *testing.T) {
	env := setupChat(t)

	if available, messages := env.history(t); available || len(messages) != 0 {
		t.Fatalf("no agent chosen: available %v, messages %v", available, messages)
	}

	if code, _ := env.ask(t, "hi"); code != http.StatusBadRequest {
		t.Fatalf("a question without an agent: %d", code)
	}

	// A provider chat is not built for yet.
	if code, body := request(t, env.e, "PUT", settingsPath, env.session, choice("anthropic", "claude-opus-5-5", "high", "sk-ant-secret-1234"), ""); code != 200 {
		t.Fatalf("settings: %d %v", code, body)
	}

	if available, _ := env.history(t); available {
		t.Fatal("chat must be unavailable for a provider without a chat profile")
	}

	if code, _ := env.ask(t, "hi"); code != http.StatusBadRequest {
		t.Fatalf("a question on an unsupported provider: %d", code)
	}
}

func TestChatStreamsStoresAndKeepsThePromptPrefixStable(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	if available, _ := env.history(t); !available {
		t.Fatal("deepseek chat must be available")
	}

	code, events := env.ask(t, "  what does it mean?  ")
	if code != http.StatusOK || len(events) < 3 || events[0].name != "delta" || events[len(events)-1].name != "done" {
		t.Fatalf("stream: %d %+v", code, events)
	}

	done := events[len(events)-1].data
	if done["question"].(map[string]any)["content"] != "what does it mean?" || done["answer"].(map[string]any)["content"] != "Answer 1." {
		t.Fatalf("done: %v", done)
	}

	if code, _ = env.ask(t, "and in a formal setting?"); code != http.StatusOK {
		t.Fatalf("second question: %d", code)
	}

	if _, messages := env.history(t); len(messages) != 4 {
		t.Fatalf("history: %v", messages)
	}

	first, second := messagesOf(env.provider.turns[0]), messagesOf(env.provider.turns[1])

	roles := ""
	for _, m := range second {
		roles += m["role"].(string)[:1]
	}

	if roles != "suau" || !strings.Contains(first[0]["content"].(string), "break a leg") {
		t.Fatalf("second request roles %q, system %q", roles, first[0]["content"])
	}

	for i := range first {
		if first[i]["role"] != second[i]["role"] || first[i]["content"] != second[i]["content"] {
			t.Fatalf("message %d changed between turns, which empties the provider cache: %v vs %v", i, first[i], second[i])
		}
	}
}

func TestAChatAboutAnEntryIsAlwaysThePlainTutor(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	// An old client may still send the switch that was removed; it changes nothing.
	req := choice("deepseek", "deepseek-v4-pro", "high", "")
	req["teacher"] = true

	if code, body := request(t, env.e, "PUT", settingsPath, env.session, req, ""); code != 200 {
		t.Fatalf("settings: %d %v", code, body)
	}

	code, body := request(t, env.e, "GET", env.path(), env.session, nil, "")
	if code != 200 || data(t, body)["teacher"].(bool) {
		t.Fatalf("a chat about an entry reports the teacher: %d %v", code, body)
	}

	if code, _ := env.ask(t, "I has a break a leg today"); code != http.StatusOK {
		t.Fatalf("question: %d", code)
	}

	system := messagesOf(env.provider.turns[0])[0]["content"].(string)
	if strings.Contains(system, "native speaker") || !strings.Contains(system, "patient tutor") {
		t.Fatalf("the chat about an entry must only answer:\n%s", system)
	}
}

func TestChatFoldsOldMessagesIntoASummaryWhenItGrows(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	// Every answer reports a prompt past the folding mark.
	env.provider.promptTokens = 8000

	for i := 1; i <= 5; i++ {
		if code, events := env.ask(t, fmt.Sprintf("question %d", i)); code != http.StatusOK {
			t.Fatalf("question %d: %d %v", i, code, events)
		}
	}

	// Five turns are ten messages: past the eight kept verbatim, so the first
	// exchange was folded by the cheap model, once, with thinking off.
	if len(env.provider.summaries) == 0 {
		t.Fatal("no summary was written")
	}

	summary := env.provider.summaries[0]
	if summary["model"] != "deepseek-flash" || summary["thinking"].(map[string]any)["type"] != "disabled" {
		t.Fatalf("summary call: %v", summary)
	}

	chat, err := env.q.GetLearningChatByEntry(context.Background(), db.GetLearningChatByEntryParams{EntryID: ids.ToPg(uuid.MustParse(env.entry)), UserID: ids.ToPg(env.user)})
	if err != nil || !strings.HasPrefix(chat.Summary, "SUMMARY") || chat.SummaryUpto == 0 {
		t.Fatalf("stored chat: %+v err %v", chat, err)
	}

	if code, _ := env.ask(t, "question 6"); code != http.StatusOK {
		t.Fatalf("question 6: %d", code)
	}

	last := messagesOf(env.provider.turns[len(env.provider.turns)-1])
	if !strings.Contains(last[0]["content"].(string), "SUMMARY") {
		t.Fatalf("the summary must reach the model: %v", last[0]["content"])
	}

	// The folded exchange is gone from the prompt but not from the history.
	for _, m := range last {
		if m["content"] == "question 1" {
			t.Fatal("a folded message was sent again")
		}
	}

	if last[1]["role"] != "user" {
		t.Fatalf("the verbatim history must start with the learner: %v", last[1])
	}

	if _, messages := env.history(t); len(messages) != 12 {
		t.Fatalf("the full history is kept: %d messages", len(messages))
	}
}

func TestChatFailedAnswerLeavesNothingBehind(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	env.provider.mu.Lock()
	env.provider.status = http.StatusServiceUnavailable
	env.provider.mu.Unlock()

	if code, events := env.ask(t, "hello?"); code != http.StatusBadGateway {
		t.Fatalf("outage: %d %v", code, events)
	}

	if _, messages := env.history(t); len(messages) != 0 {
		t.Fatalf("a failed turn must not be stored: %v", messages)
	}

	env.provider.mu.Lock()
	env.provider.status = http.StatusOK
	env.provider.mu.Unlock()

	if code, _ := env.ask(t, "hello?"); code != http.StatusOK {
		t.Fatalf("retry after the outage: %d", code)
	}
}

func TestChatAllowsOneTurnAtATimeAndCanBeCleared(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	if code, _ := env.ask(t, "first"); code != http.StatusOK {
		t.Fatalf("first: %d", code)
	}

	chat, err := env.q.GetLearningChatByEntry(context.Background(), db.GetLearningChatByEntryParams{EntryID: ids.ToPg(uuid.MustParse(env.entry)), UserID: ids.ToPg(env.user)})
	if err != nil || chat.BusyUntil.Valid {
		t.Fatalf("the chat must be free after a turn: %+v %v", chat, err)
	}

	now := time.Now().UTC()
	if _, err := env.q.LockLearningChat(context.Background(), db.LockLearningChatParams{ID: chat.ID, UserID: chat.UserID, BusyUntil: db.Time(now.Add(time.Minute)), Now: db.Time(now)}); err != nil {
		t.Fatal(err)
	}

	if code, _ := env.ask(t, "second"); code != http.StatusConflict {
		t.Fatalf("a busy chat: %d", code)
	}

	if code, _ := request(t, env.e, "DELETE", env.path(), env.session, nil, ""); code != http.StatusConflict {
		t.Fatalf("clearing a busy chat: %d", code)
	}

	if err := env.q.UnlockLearningChat(context.Background(), chat.ID); err != nil {
		t.Fatal(err)
	}

	if code, _ := request(t, env.e, "DELETE", env.path(), env.session, nil, ""); code != http.StatusOK {
		t.Fatalf("clear: %d", code)
	}

	if _, messages := env.history(t); len(messages) != 0 {
		t.Fatalf("cleared chat: %v", messages)
	}
}

func TestChatBelongsToTheEntrysOwnerAndChecksTheText(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	for name, text := range map[string]string{"empty": "   ", "too long": strings.Repeat("я", 2001)} {
		if code, _ := env.ask(t, text); code != http.StatusBadRequest {
			t.Errorf("%s: %d", name, code)
		}
	}

	pool, q := testdb.Pool(t)
	stranger := testdb.NewUser(t, pool, q, time.Now().UTC(), 2)
	other := env
	other.session = token.NewCodec(authSecret).Issue(stranger, time.Now().UTC(), time.Hour)

	if code, _ := other.ask(t, "hi"); code != http.StatusNotFound {
		t.Fatalf("someone else's entry: %d", code)
	}

	if code, _ := request(t, other.e, "GET", other.path(), other.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("someone else's chat: %d", code)
	}
}

func TestChatStopsAtItsTokenBudgetAndClearingDoesNotGiveItBack(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	// Each answer costs 70 005 tokens; the limit is 120 000. The turn that
	// crosses it still completes, the next one is refused.
	env.provider.promptTokens = 70000

	var done map[string]any

	for i := 1; i <= 2; i++ {
		code, events := env.ask(t, fmt.Sprintf("question %d", i))
		if code != http.StatusOK {
			t.Fatalf("question %d: %d %v", i, code, events)
		}

		done = events[len(events)-1].data["budget"].(map[string]any)
	}

	if done["limit"] != float64(120000) || done["used"] != float64(140010) || done["resetsAt"] == nil {
		t.Fatalf("budget after two turns: %v", done)
	}

	rec := env.post(t, "question 3")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over budget: %d retry-after %q body %s", rec.Code, rec.Header().Get("Retry-After"), rec.Body)
	}

	var refused map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &refused)

	if refused["code"] != float64(42902) || len(refused["details"].([]any)) != 1 {
		t.Fatalf("refusal: %v", refused)
	}

	if len(env.provider.turns) != 2 {
		t.Fatalf("a refused question must not reach the provider: %d calls", len(env.provider.turns))
	}

	code, body := request(t, env.e, "GET", env.path(), env.session, nil, "")
	budget := data(t, body)["budget"].(map[string]any)

	if code != 200 || budget["used"] != float64(140010) {
		t.Fatalf("get chat: %d %v", code, body)
	}

	// Starting the conversation over must not buy a new budget.
	if code, _ := request(t, env.e, "DELETE", env.path(), env.session, nil, ""); code != http.StatusOK {
		t.Fatalf("clear: %d", code)
	}

	if code, _ := env.ask(t, "again"); code != http.StatusTooManyRequests {
		t.Fatalf("a cleared chat is still over budget: %d", code)
	}

	// Once the window has ended the chat is whole again, and counts from zero.
	_, err := env.pool.Exec(context.Background(), "UPDATE learning_chats SET spent_since = now() - interval '25 hours' WHERE user_id = $1", env.user)
	if err != nil {
		t.Fatal(err)
	}

	env.provider.promptTokens = 300

	code, events := env.ask(t, "a new day")
	if code != http.StatusOK {
		t.Fatalf("after the window: %d %v", code, events)
	}

	if used := events[len(events)-1].data["budget"].(map[string]any)["used"]; used != float64(305) {
		t.Fatalf("the new window must count from zero: %v", used)
	}
}

func TestChatRateLimitIsPerUserAnd429(t *testing.T) {
	env := setupChatRate(t, ratelimit.Config{Burst: 2, Every: time.Hour, InFlight: 2})
	env.useDeepSeek(t)

	for i := 1; i <= 2; i++ {
		if code, _ := env.ask(t, "question"); code != http.StatusOK {
			t.Fatalf("question %d inside the burst: %d", i, code)
		}
	}

	rec := env.post(t, "question")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over the rate: %d retry-after %q", rec.Code, rec.Header().Get("Retry-After"))
	}

	var refused map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &refused)

	if refused["code"] != float64(42901) || len(env.provider.turns) != 2 {
		t.Fatalf("refusal %v after %d provider calls", refused, len(env.provider.turns))
	}

	// Someone else is not held back by it.
	pool, q := testdb.Pool(t)
	other := env
	other.user = testdb.NewUser(t, pool, q, time.Now().UTC(), 3)
	other.session = token.NewCodec(authSecret).Issue(other.user, time.Now().UTC(), time.Hour)

	if code, _ := request(t, other.e, "GET", other.path(), other.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("reading is not limited, and the entry is not theirs: %d", code)
	}
}

func TestChatDailyLimitBelongsToTheUserNotTheKey(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	env.setUsage(t, "user", env.user.String(), 0, 400000)

	rec := env.post(t, "question")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over the daily limit: %d retry-after %q", rec.Code, rec.Header().Get("Retry-After"))
	}

	var refused map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &refused)

	if refused["code"] != float64(42903) || len(refused["details"].([]any)) != 1 || len(env.provider.turns) != 0 {
		t.Fatalf("refusal %v after %d provider calls", refused, len(env.provider.turns))
	}

	code, body := request(t, env.e, "GET", env.path(), env.session, nil, "")
	daily := data(t, body)["daily"].(map[string]any)

	if code != 200 || daily["used"] != float64(400000) || daily["limit"] != float64(400000) {
		t.Fatalf("daily in the chat: %d %v", code, daily)
	}

	// A different API key is the same user: nothing is gained.
	env.useKey(t, "sk-deepseek-another-key")

	if code, _ := env.ask(t, "question"); code != http.StatusTooManyRequests {
		t.Fatalf("a new key must not lift the user's limit: %d", code)
	}
}

func TestChatDailyCountersStartOverEachDayAndAreSwept(t *testing.T) {
	env := setupChat(t)
	env.useDeepSeek(t)

	// Yesterday's spending does not count today, and a week-old row is swept
	// by the next turn.
	env.setUsage(t, "user", env.user.String(), 1, 999999)
	env.setUsage(t, "user", env.user.String(), 10, 1)

	code, events := env.ask(t, "question")
	if code != http.StatusOK {
		t.Fatalf("a new day: %d %v", code, events)
	}

	if daily := events[len(events)-1].data["daily"].(map[string]any); daily["used"] != float64(305) {
		t.Fatalf("today counts from zero: %v", daily)
	}

	var old int
	err := env.pool.QueryRow(context.Background(), "SELECT count(*) FROM learning_chat_usage WHERE subject = $1 AND day < (now() AT TIME ZONE 'utc')::date - 7", env.user.String()).Scan(&old)
	if err != nil || old != 0 {
		t.Fatalf("rows older than a week must be swept: %d %v", old, err)
	}
}

func TestChatDailyLimitOfAKeyIsSharedByEveryAccountUsingIt(t *testing.T) {
	first, second := setupChat(t), setupChat(t)
	first.useDeepSeek(t)
	second.useDeepSeek(t)

	// The key has been used up by others today.
	fingerprint := secretbox.New(authSecret).Fingerprint([]byte(defaultKey))
	first.setUsage(t, "key", fingerprint, 0, 800000)

	for name, env := range map[string]chatEnv{"first": first, "second": second} {
		if code, _ := env.ask(t, "question"); code != http.StatusTooManyRequests {
			t.Errorf("%s account on the used-up key: %d", name, code)
		}
	}

	// The second account brings its own key and is free to go on; the first is not.
	second.useKey(t, "sk-deepseek-second-own-key")

	if code, _ := second.ask(t, "question"); code != http.StatusOK {
		t.Fatalf("an account with its own key: %d", code)
	}

	if code, _ := first.ask(t, "question"); code != http.StatusTooManyRequests {
		t.Fatalf("the first account is still on the used-up key: %d", code)
	}

	// What is stored is the fingerprint, never the key.
	var leaked int
	err := first.pool.QueryRow(context.Background(), "SELECT count(*) FROM learning_chat_usage WHERE subject LIKE '%sk-deepseek%'").Scan(&leaked)
	if err != nil || leaked != 0 {
		t.Fatalf("an API key reached the usage table: %d %v", leaked, err)
	}
}
