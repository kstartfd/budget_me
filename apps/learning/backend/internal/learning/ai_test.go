package learning_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
	"learning_app/backend/internal/token"
)

const settingsPath = "/api/v1/learning/ai/settings"

func choice(provider, model, level, key string) map[string]any {
	return map[string]any{"provider": provider, "model": model, "level": level, "apiKey": key}
}

func TestAISettingsAreValidatedSealedAndWriteOnly(t *testing.T) {
	env := setupAI(t)

	code, body := request(t, env.e, "GET", settingsPath, env.session, nil, "")
	if code != 200 || body["data"] != nil {
		t.Fatalf("nothing chosen yet: %d %v", code, body)
	}

	for name, req := range map[string]map[string]any{
		"unknown provider":    choice("nobody", "x", "low", "sk-test-key-1234"),
		"model of another":    choice("openai", "claude-opus-5-5", "low", "sk-test-key-1234"),
		"level it lacks":      choice("anthropic", "claude-opus-5-5", "none", "sk-test-key-1234"),
		"too short":           choice("anthropic", "claude-opus-5-5", "high", "short"),
		"no key stored yet":   choice("anthropic", "claude-opus-5-5", "high", ""),
		"key only whitespace": choice("anthropic", "claude-opus-5-5", "high", "        "),
	} {
		if code, body = request(t, env.e, "PUT", settingsPath, env.session, req, ""); code != 400 {
			t.Errorf("%s: %d %v", name, code, body)
		}
	}

	code, body = request(t, env.e, "PUT", settingsPath, env.session, choice("anthropic", "claude-opus-5-5", "high", "  sk-ant-secret-WXYZ  "), "")
	if code != 200 {
		t.Fatalf("save: %d %v", code, body)
	}

	saved := data(t, body)
	if saved["provider"] != "anthropic" || saved["model"] != "claude-opus-5-5" || saved["level"] != "high" || saved["keyHint"] != "WXYZ" {
		t.Fatalf("saved: %v", saved)
	}

	if strings.Contains(fmt.Sprint(body), "secret") {
		t.Fatalf("response carries the key: %v", body)
	}

	row, err := env.q.GetLearningAISettings(context.Background(), ids.ToPg(env.user))
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(row.KeyCiphertext, []byte("sk-ant-secret")) {
		t.Fatal("key stored in the clear")
	}

	opened, err := secretbox.New(authSecret).Open(row.KeyCiphertext, env.user[:])
	if err != nil || string(opened) != "sk-ant-secret-WXYZ" {
		t.Fatalf("sealed key: %q %v", opened, err)
	}

	code, body = request(t, env.e, "PUT", settingsPath, env.session, choice("anthropic", "claude-haiku-5-5", "low", ""), "")
	if code != 200 || data(t, body)["model"] != "claude-haiku-5-5" || data(t, body)["keyHint"] != "WXYZ" {
		t.Fatalf("change the agent, keep the key: %d %v", code, body)
	}

	code, body = request(t, env.e, "PUT", settingsPath, env.session, choice("openai", "gpt-6.1-terra", "none", ""), "")
	if code != 400 {
		t.Fatalf("a key must not follow the user to another provider: %d %v", code, body)
	}

	code, body = request(t, env.e, "GET", settingsPath, env.session, nil, "")
	if code != 200 || data(t, body)["model"] != "claude-haiku-5-5" {
		t.Fatalf("get: %d %v", code, body)
	}
}

func TestAISettingsRejectAKeyTheProviderRefuses(t *testing.T) {
	env := setupAI(t)
	env.keyStatus.Store(http.StatusUnauthorized)

	code, body := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "high", "sk-not-a-real-key"), "")
	if code != 400 {
		t.Fatalf("rejected key saved: %d %v", code, body)
	}

	if _, err := env.q.GetLearningAISettings(context.Background(), ids.ToPg(env.user)); !db.IsNotFound(err) {
		t.Fatalf("a rejected key left a row behind: %v", err)
	}

	env.keyStatus.Store(http.StatusServiceUnavailable)

	code, body = request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "high", "sk-not-a-real-key"), "")
	if code != 200 {
		t.Fatalf("an outage at the provider must not block saving: %d %v", code, body)
	}
}

func TestAISettingsBelongToTheSessionOnly(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	_, extension := apiToken(t, env.e, env.session, "read_write", topicID)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("openai", "gpt-6-luna", "low", "sk-test-key-1234"), ""); code != 200 {
		t.Fatalf("save: %d", code)
	}

	for _, method := range []string{"GET", "PUT"} {
		if code, _ := request(t, env.e, method, settingsPath, extension, choice("openai", "gpt-6-luna", "low", "sk-other-key-5678"), ""); code != 401 {
			t.Errorf("extension token %s reached the agent settings: %d", method, code)
		}
	}

	stranger := token.NewCodec(authSecret).Issue(ids.New(), time.Now().UTC(), time.Hour)

	code, body := request(t, env.e, "GET", settingsPath, stranger, nil, "")
	if code != 200 && code != 401 {
		t.Fatalf("stranger: %d %v", code, body)
	}

	if code == 200 && body["data"] != nil {
		t.Fatalf("another account's settings leaked: %v", body)
	}
}

// capture enqueues one capture as the user and runs the queue once.
func capture(t *testing.T, env aiEnv, topicID string, fields map[string]any) db.LearningEntry {
	t.Helper()

	fields["topicId"] = topicID

	code, body := request(t, env.e, "POST", "/api/v1/learning/captures", env.session, fields, "")
	if code != 202 {
		t.Fatalf("enqueue: %d %v", code, body)
	}

	jobID := data(t, body)["id"].(string)

	if worked, err := env.processor.Handle(context.Background()); !worked || err != nil {
		t.Fatalf("process: %t %v", worked, err)
	}

	code, body = request(t, env.e, "GET", "/api/v1/learning/captures/"+jobID, env.session, nil, "")
	if code != 200 || data(t, body)["status"] != "done" {
		t.Fatalf("capture: %d %v", code, body)
	}

	entry, err := env.q.GetLearningEntry(context.Background(), db.GetLearningEntryParams{ID: ids.ToPg(uuid.MustParse(data(t, body)["entryId"].(string))), UserID: ids.ToPg(env.user)})
	if err != nil {
		t.Fatal(err)
	}

	return entry
}

func chooseAgent(t *testing.T, env aiEnv) {
	t.Helper()

	if code, body := request(t, env.e, "PUT", settingsPath, env.session, choice("anthropic", "claude-opus-5-5", "high", "sk-ant-test-key-1234"), ""); code != 200 {
		t.Fatalf("choose: %d %v", code, body)
	}
}

func TestCaptureIsWrittenByTheChosenAgent(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)

	env.agent.set("```json\n{\"transcription\":\"/ˌserənˈdɪpɪti/\",\"translation\":\"счастливая случайность\",\"explanation\":\"serendipity — находка\",\"nativeUsage\":\"formal\",\"example\":\"Pure serendipity.\",\"rule\":\"noun\",\"ignored\":\"x\"}\n```", nil)

	entry := capture(t, env, topicID, map[string]any{"kind": "word", "text": "serendipity", "source": "Futurama"})
	if entry.Translation != "счастливая случайность" || entry.Transcription != "/ˌserənˈdɪpɪti/" || entry.Explanation != "serendipity — находка" || entry.NativeUsage != "formal" || entry.Example != "Pure serendipity." || entry.Rule != "noun" {
		t.Fatalf("entry not written: %+v", entry)
	}

	calls := env.agent.requests()
	if len(calls) != 1 {
		t.Fatalf("agent called %d times", len(calls))
	}

	call := calls[0]
	if call.Provider != llm.Anthropic || call.Model != "claude-opus-5-5" || call.Level != "high" || call.Key != "sk-ant-test-key-1234" {
		t.Fatalf("request: %+v", call)
	}

	if !strings.Contains(call.System, "English") || !strings.Contains(call.System, "Russian") || !strings.Contains(call.Prompt, "serendipity") || !strings.Contains(call.Prompt, "Futurama") || strings.Contains(call.System, "serendipity") || strings.Contains(call.System, "Futurama") {
		t.Fatalf("prompt lost the topic or the text:\n%s\n%s", call.System, call.Prompt)
	}
}

func TestAgentOnlyWritesWhatTheCaptureLeftEmpty(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)

	env.agent.set(`{"translation":"агент","explanation":"агент","rule":"агент"}`, nil)

	entry := capture(t, env, topicID, map[string]any{"text": "hit the road", "translation": "отправиться в путь"})
	if entry.Translation != "отправиться в путь" || entry.Explanation != "агент" {
		t.Fatalf("a supplied field was overwritten or an empty one skipped: %+v", entry)
	}

	if strings.Contains(env.agent.requests()[0].Prompt, `"translation"`) {
		t.Fatalf("asked for a field that was already there: %s", env.agent.requests()[0].Prompt)
	}

	env.agent.set(`{}`, nil)

	before := len(env.agent.requests())

	capture(t, env, topicID, map[string]any{
		"text": "full card", "transcription": "a", "translation": "b", "explanation": "c", "nativeUsage": "d", "example": "e", "rule": "f",
	})

	if len(env.agent.requests()) != before {
		t.Fatal("a complete card still went to the agent")
	}
}

func TestCaptureWithoutAnAgentIsLeftAsSent(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)

	entry := capture(t, env, topicID, map[string]any{"text": "no agent here"})
	if entry.Translation != "" || len(env.agent.requests()) != 0 {
		t.Fatalf("enriched without a chosen agent: %+v", entry)
	}
}

func TestFailingAgentNeverLosesTheCapture(t *testing.T) {
	for name, reply := range map[string]struct {
		text string
		err  error
	}{
		"permanent failure": {"", errors.New("provider status 400")},
		"rejected key":      {"", llm.ErrRejected},
		"unreadable reply":  {"Sure! Here is the translation.", nil},
		"wrong shape":       {`{"translation":["not","a","string"]}`, nil},
	} {
		env := setupAI(t)
		topicID := topic(t, env.e, env.session)
		chooseAgent(t, env)
		env.agent.set(reply.text, reply.err)

		entry := capture(t, env, topicID, map[string]any{"text": name})
		if entry.Text != name || entry.Translation != "" {
			t.Errorf("%s: %+v", name, entry)
		}
	}
}

func TestTemporaryAgentFailureRetriesThenSavesAsSent(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)
	env.agent.set("", fmt.Errorf("%w: provider status 429", llm.ErrTemporary))

	code, body := request(t, env.e, "POST", "/api/v1/learning/captures", env.session, map[string]any{"topicId": topicID, "text": "rate limited"}, "")
	if code != 202 {
		t.Fatalf("enqueue: %d %v", code, body)
	}

	jobID := uuid.MustParse(data(t, body)["id"].(string))

	if _, err := env.processor.Handle(context.Background()); err == nil {
		t.Fatal("a temporary failure on the first attempt must be retried")
	}

	job, err := env.q.GetLearningCapture(context.Background(), db.GetLearningCaptureParams{ID: ids.ToPg(jobID), UserID: ids.ToPg(env.user)})
	if err != nil || job.Status != db.LearningCaptureStatusPending || job.Attempts != 1 {
		t.Fatalf("first attempt: %+v %v", job, err)
	}

	if _, err := env.pool.Exec(context.Background(), "UPDATE learning_captures SET attempts = 2, available_at = $2 WHERE id = $1", jobID, time.Now().UTC().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}

	if worked, err := env.processor.Handle(context.Background()); !worked || err != nil {
		t.Fatalf("the last attempt must save the capture as it came: %t %v", worked, err)
	}

	job, err = env.q.GetLearningCapture(context.Background(), db.GetLearningCaptureParams{ID: ids.ToPg(jobID), UserID: ids.ToPg(env.user)})
	if err != nil || job.Status != db.LearningCaptureStatusDone || job.Attempts != 3 {
		t.Fatalf("third attempt: %+v %v", job, err)
	}
}

func TestDisconnectingForgetsTheKeyAndStopsTheAgent(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)

	stranger := token.NewCodec(authSecret).Issue(ids.New(), time.Now().UTC(), time.Hour)
	if code, _ := request(t, env.e, "DELETE", settingsPath, stranger, nil, ""); code != 200 {
		t.Fatalf("another account's disconnect: %d", code)
	}

	if _, err := env.q.GetLearningAISettings(context.Background(), ids.ToPg(env.user)); err != nil {
		t.Fatalf("someone else's disconnect removed this user's agent: %v", err)
	}

	if code, body := request(t, env.e, "DELETE", settingsPath, env.session, nil, ""); code != 200 {
		t.Fatalf("disconnect: %d %v", code, body)
	}

	if code, body := request(t, env.e, "DELETE", settingsPath, env.session, nil, ""); code != 200 {
		t.Fatalf("disconnecting twice is not an error: %d %v", code, body)
	}

	if _, err := env.q.GetLearningAISettings(context.Background(), ids.ToPg(env.user)); !db.IsNotFound(err) {
		t.Fatalf("the key outlived the disconnect: %v", err)
	}

	if code, body := request(t, env.e, "GET", settingsPath, env.session, nil, ""); code != 200 || body["data"] != nil {
		t.Fatalf("get after disconnect: %d %v", code, body)
	}

	env.agent.set(`{"translation":"nope"}`, nil)

	if entry := capture(t, env, topicID, map[string]any{"text": "after disconnect"}); entry.Translation != "" || len(env.agent.requests()) != 0 {
		t.Fatalf("a disconnected agent still wrote: %+v", entry)
	}
}
