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
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
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
	provider *deepseek
	user     uuid.UUID
	session  string
	entry    string
}

func setupChat(t *testing.T) chatEnv {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)

	provider := &deepseek{status: http.StatusOK, promptTokens: 300}
	srv := httptest.NewServer(provider)
	t.Cleanup(srv.Close)

	cfg := config.Config{AuthSecret: authSecret, LLMURLs: llm.URLs{Anthropic: srv.URL, OpenAI: srv.URL, DeepSeek: srv.URL}}
	e := server.New(cfg, pool, testdb.AccountPool(t))
	session := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	topicID := topic(t, e, session)

	code, body := request(t, e, "POST", "/api/v1/learning/entries", session, map[string]any{"topicId": topicID, "kind": "phrase", "text": "break a leg", "status": "learning"}, "")
	if code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	return chatEnv{e: e, q: q, provider: provider, user: user, session: session, entry: data(t, body)["id"].(string)}
}

func (c chatEnv) path() string { return "/api/v1/learning/entries/" + c.entry + "/chat" }

func (c chatEnv) useDeepSeek(t *testing.T) {
	t.Helper()

	if code, body := request(t, c.e, "PUT", settingsPath, c.session, choice("deepseek", "deepseek-v4-pro", "high", "sk-deepseek-1234"), ""); code != 200 {
		t.Fatalf("settings: %d %v", code, body)
	}
}

type sseEvent struct {
	name string
	data map[string]any
}

// ask posts a question and returns the status with the parsed event stream (or,
// for a refusal, the JSON envelope as one event named "body").
func (c chatEnv) ask(t *testing.T, text string) (int, []sseEvent) {
	t.Helper()

	raw, _ := json.Marshal(map[string]any{"text": text})
	req := httptest.NewRequest(http.MethodPost, c.path(), bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.session)

	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)

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
