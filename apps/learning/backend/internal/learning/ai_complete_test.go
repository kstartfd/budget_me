package learning_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/secretbox"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

const completePath = "/api/v1/learning/ai/complete"

func prompt(system, text string) map[string]any {
	return map[string]any{"system": system, "prompt": text}
}

func TestAICompleteRunsThePromptThroughTheUsersAgent(t *testing.T) {
	env := setupAI(t)
	env.provider.reply("Привет, мир")

	if code, body := request(t, env.e, "POST", completePath, env.session, prompt("Be brief.", "hello"), ""); code != 404 {
		t.Fatalf("no agent chosen yet: %d %v", code, body)
	}

	for _, agent := range []struct {
		provider, model, level, key string
		path                        string
		sent                        func(map[string]any) bool
	}{
		{
			"anthropic", "claude-opus-5-5", "high", "sk-ant-secret-WXYZ", "/v1/messages",
			func(b map[string]any) bool {
				effort, _ := b["output_config"].(map[string]any)["effort"].(string)

				return b["model"] == "claude-opus-5-5" && effort == "high"
			},
		},
		{
			"openai", "gpt-6.1-terra", "none", "sk-openai-secret-ABCD", "/responses",
			func(b map[string]any) bool {
				effort, _ := b["reasoning"].(map[string]any)["effort"].(string)

				return b["model"] == "gpt-6.1-terra" && effort == "none" && b["instructions"] == "Be brief."
			},
		},
		{
			"deepseek", "deepseek-flash", "off", "sk-deepseek-secret-EFGH", "/chat/completions",
			func(b map[string]any) bool {
				thinking, _ := b["thinking"].(map[string]any)["type"].(string)

				return b["model"] == "deepseek-flash" && thinking == "disabled"
			},
		},
	} {
		if code, body := request(t, env.e, "PUT", settingsPath, env.session, choice(agent.provider, agent.model, agent.level, agent.key), ""); code != 200 {
			t.Fatalf("%s: save: %d %v", agent.provider, code, body)
		}

		before := len(env.provider.requests())

		code, body := request(t, env.e, "POST", completePath, env.session, prompt("Be brief.", "hello"), "")
		if code != 200 || data(t, body)["text"] != "Привет, мир" {
			t.Fatalf("%s: complete: %d %v", agent.provider, code, body)
		}

		if strings.Contains(string(mustJSON(t, body)), "secret") {
			t.Fatalf("%s: the response carries the key: %v", agent.provider, body)
		}

		calls := env.provider.requests()[before:]
		if len(calls) != 1 {
			t.Fatalf("%s: %d provider calls, want 1", agent.provider, len(calls))
		}

		call := calls[0]
		if !strings.HasSuffix(call.Path, agent.path) || call.Key != agent.key {
			t.Errorf("%s: sent to %s with %q", agent.provider, call.Path, call.Key)
		}

		var sent map[string]any
		if err := json.Unmarshal([]byte(call.Body), &sent); err != nil || !agent.sent(sent) || !strings.Contains(call.Body, "hello") {
			t.Errorf("%s: unexpected provider request: %s", agent.provider, call.Body)
		}
	}
}

func TestAICompleteBelongsToTheSessionOnly(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	_, extension := apiToken(t, env.e, env.session, "read_write", topicID)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("openai", "gpt-6-luna", "low", "sk-test-key-1234"), ""); code != 200 {
		t.Fatalf("save: %d", code)
	}

	if code, _ := request(t, env.e, "POST", completePath, extension, prompt("", "hello"), ""); code != 401 {
		t.Errorf("an extension token spent the user's key: %d", code)
	}

	if code, _ := request(t, env.e, "POST", completePath, "", prompt("", "hello"), ""); code != 401 {
		t.Errorf("no credentials at all: %d", code)
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("the provider was called %d times for refused requests", len(calls))
	}

	// Another account never reaches this user's agent: it has none of its own.
	stranger := testdb.NewUser(t, env.pool, env.q, time.Now().UTC(), 2)
	session := token.NewCodec(authSecret).Issue(stranger, time.Now().UTC(), time.Hour)

	if code, body := request(t, env.e, "POST", completePath, session, prompt("", "hello"), ""); code != 404 {
		t.Fatalf("another account used this agent: %d %v", code, body)
	}
}

func TestAICompleteChecksTheRequest(t *testing.T) {
	env := setupAI(t)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("anthropic", "claude-haiku-5-5", "low", "sk-ant-test-key-1234"), ""); code != 200 {
		t.Fatalf("save: %d", code)
	}

	for name, req := range map[string]map[string]any{
		"no prompt":          {"system": "x"},
		"blank prompt":       prompt("x", "  \n "),
		"prompt too long":    prompt("", strings.Repeat("я", 60_001)),
		"system too long":    prompt(strings.Repeat("я", 10_001), "hello"),
		"wrong type of body": {"prompt": 7},
	} {
		if code, body := request(t, env.e, "POST", completePath, env.session, req, ""); code != 400 {
			t.Errorf("%s: %d %v", name, code, body)
		}
	}

	// The longest prompt that is allowed goes through.
	env.provider.reply("ok")

	if code, body := request(t, env.e, "POST", completePath, env.session, prompt(strings.Repeat("я", 10_000), strings.Repeat("я", 60_000)), ""); code != 200 {
		t.Fatalf("a prompt at the limit: %d %v", code, body)
	}

	// Past the byte cap the body is refused before it is read.
	oversized, err := json.Marshal(prompt("", strings.Repeat("a", 200*1024)))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", completePath, bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.session)

	rec := httptest.NewRecorder()
	env.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: %d", rec.Code)
	}

	if calls := env.provider.requests(); len(calls) != 1 {
		t.Fatalf("refused requests reached the provider: %d calls", len(calls))
	}
}

func TestAICompleteSaysWhatWentWrongAtTheProvider(t *testing.T) {
	env := setupAI(t)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "high", "sk-test-key-1234"), ""); code != 200 {
		t.Fatalf("save: %d", code)
	}

	for _, tc := range []struct {
		name     string
		status   int32
		reply    string
		wantHTTP int
		wantCode float64
	}{
		{"key refused", http.StatusUnauthorized, "x", 422, 30001},
		{"outage", http.StatusServiceUnavailable, "x", 503, 30002},
		{"rate limit", http.StatusTooManyRequests, "x", 503, 30002},
		{"rejected request", http.StatusBadRequest, "x", 502, 30003},
		{"empty answer", http.StatusOK, "", 502, 30003},
	} {
		env.keyStatus.Store(tc.status)
		env.provider.reply(tc.reply)

		code, body := request(t, env.e, "POST", completePath, env.session, prompt("", "hello"), "")
		if code != tc.wantHTTP || body["code"] != tc.wantCode {
			t.Errorf("%s: %d %v, want %d / %v", tc.name, code, body, tc.wantHTTP, tc.wantCode)
		}

		if strings.Contains(string(mustJSON(t, body)), "sk-test-key") {
			t.Errorf("%s: the error carries the key: %v", tc.name, body)
		}
	}
}

func TestAICompleteAsksForTheKeyAgainWhenItCannotBeOpened(t *testing.T) {
	env := setupAI(t)

	// Sealed with another secret, as after AUTH_SECRET was rotated.
	sealed, err := secretbox.New("some-other-secret-that-is-long-enough").Seal([]byte("sk-old-key-1234"), env.user[:])
	if err != nil {
		t.Fatal(err)
	}

	if _, err := env.q.UpsertLearningAISettings(context.Background(), db.UpsertLearningAISettingsParams{
		UserID: ids.ToPg(env.user), Provider: db.LearningAiProviderOpenai, Model: "gpt-6-luna", Level: db.LearningAiLevelLow,
		KeyCiphertext: sealed, KeyHint: "1234", UpdatedAt: db.Time(time.Now().UTC()),
	}); err != nil {
		t.Fatal(err)
	}

	if code, body := request(t, env.e, "POST", completePath, env.session, prompt("", "hello"), ""); code != 409 {
		t.Fatalf("unreadable key: %d %v", code, body)
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("the provider was called without a key: %d", len(calls))
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return raw
}
