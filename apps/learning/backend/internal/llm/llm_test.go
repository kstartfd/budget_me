package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning_app/backend/internal/llm"
)

type seen struct {
	path   string
	header http.Header
	body   map[string]any
}

func provider(t *testing.T, status int, reply string) (llm.Client, *seen) {
	t.Helper()

	got := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.header = r.Header

		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&got.body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)

	return llm.New(llm.URLs{Anthropic: srv.URL, OpenAI: srv.URL, DeepSeek: srv.URL, Kimi: srv.URL}), got
}

const claudeReply = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":"привет"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`

func TestAnthropicRequestShapeAndText(t *testing.T) {
	c, got := provider(t, 200, claudeReply)

	text, err := c.Complete(context.Background(), llm.Request{Provider: llm.Anthropic, Model: "claude-opus-5-5", Level: "xhigh", Key: "sk-ant-secret", System: "sys", Prompt: "hi"})
	if err != nil || text != "привет" {
		t.Fatalf("text %q err %v", text, err)
	}

	if got.path != "/v1/messages" || got.header.Get("Authorization") != "Bearer sk-ant-secret" {
		t.Fatalf("path %q auth %q", got.path, got.header.Get("Authorization"))
	}

	if got.body["model"] != "claude-opus-5-5" || got.body["output_config"].(map[string]any)["effort"] != "xhigh" {
		t.Fatalf("body %v", got.body)
	}

	if _, has := got.body["thinking"]; has {
		t.Fatalf("thinking must be left to the model, got %v", got.body["thinking"])
	}
}

func TestAnthropicRefusalIsAnErrorNotRetried(t *testing.T) {
	c, _ := provider(t, 200, strings.Replace(claudeReply, "end_turn", "refusal", 1))

	_, err := c.Complete(context.Background(), llm.Request{Provider: llm.Anthropic, Model: "claude-opus-5-5", Level: "low", Key: "k", System: "s", Prompt: "p"})
	if err == nil || errors.Is(err, llm.ErrTemporary) {
		t.Fatalf("want permanent error, got %v", err)
	}
}

func TestOpenAIRequestShapeAndText(t *testing.T) {
	c, got := provider(t, 200, `{"output":[{"type":"reasoning","content":[]},{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`)

	text, err := c.Complete(context.Background(), llm.Request{Provider: llm.OpenAI, Model: "gpt-6.1-terra", Level: "none", Key: "sk-secret", System: "sys", Prompt: "hi"})
	if err != nil || text != "hello" {
		t.Fatalf("text %q err %v", text, err)
	}

	if got.path != "/responses" || got.header.Get("Authorization") != "Bearer sk-secret" {
		t.Fatalf("path %q auth %q", got.path, got.header.Get("Authorization"))
	}

	if got.body["model"] != "gpt-6.1-terra" || got.body["instructions"] != "sys" || got.body["input"] != "hi" || got.body["reasoning"].(map[string]any)["effort"] != "none" {
		t.Fatalf("body %v", got.body)
	}
}

func TestDeepSeekLevelsMapOntoThinking(t *testing.T) {
	reply := `{"choices":[{"message":{"content":"ok"}}]}`

	for level, want := range map[string]struct{ thinking, effort string }{
		"off":  {"disabled", ""},
		"low":  {"enabled", "low"},
		"high": {"enabled", "high"},
		"max":  {"enabled", "max"},
	} {
		c, got := provider(t, 200, reply)

		if _, err := c.Complete(context.Background(), llm.Request{Provider: llm.DeepSeek, Model: "deepseek-flash", Level: level, Key: "sk-secret", System: "s", Prompt: "p"}); err != nil {
			t.Fatal(err)
		}

		effort, _ := got.body["reasoning_effort"].(string)
		if got.path != "/chat/completions" || got.body["thinking"].(map[string]any)["type"] != want.thinking || effort != want.effort {
			t.Fatalf("%s: path %q body %v", level, got.path, got.body)
		}
	}
}

func TestKimiRequestShapeAndText(t *testing.T) {
	c, got := provider(t, 200, `{"choices":[{"message":{"reasoning_content":"thinking about it","content":"привет"}}]}`)

	text, err := c.Complete(context.Background(), llm.Request{Provider: llm.Kimi, Model: "kimi-k3.1-pro", Level: "high", Key: "sk-moon-secret", System: "sys", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}

	if text != "привет" {
		t.Fatalf("only the answer is kept, not the reasoning: %q", text)
	}

	if got.path != "/chat/completions" || got.header.Get("Authorization") != "Bearer sk-moon-secret" {
		t.Fatalf("path %q header %v", got.path, got.header)
	}

	if got.body["model"] != "kimi-k3.1-pro" || got.body["reasoning_effort"] != "high" || got.body["max_completion_tokens"] != float64(32000) {
		t.Fatalf("body: %v", got.body)
	}

	for _, key := range []string{"thinking", "max_tokens", "temperature"} {
		if _, ok := got.body[key]; ok {
			t.Fatalf("%s must not be sent to K3: %v", key, got.body)
		}
	}

	msgs := got.body["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[1].(map[string]any)["content"] != "hi" {
		t.Fatalf("messages: %v", msgs)
	}
}

func TestKimiAnswerWithNoTextIsAnError(t *testing.T) {
	c, _ := provider(t, 200, `{"choices":[{"message":{"reasoning_content":"only thoughts","content":"  "}}]}`)

	if _, err := c.Complete(context.Background(), llm.Request{Provider: llm.Kimi, Model: "kimi-k3", Level: "low", Key: "k", System: "s", Prompt: "p"}); err == nil {
		t.Fatal("an answer with no text must be an error")
	}
}

func TestProviderStatusesAreSortedForCallers(t *testing.T) {
	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek, llm.Kimi} {
		for status, check := range map[int]func(error) bool{
			401: func(err error) bool { return errors.Is(err, llm.ErrRejected) },
			429: func(err error) bool { return errors.Is(err, llm.ErrTemporary) },
			500: func(err error) bool { return errors.Is(err, llm.ErrTemporary) },
			400: func(err error) bool {
				return err != nil && !errors.Is(err, llm.ErrTemporary) && !errors.Is(err, llm.ErrRejected)
			},
		} {
			c, _ := provider(t, status, `{"error":{"message":"nope"}}`)

			_, err := c.Complete(context.Background(), llm.Request{Provider: p, Model: "m", Level: "low", Key: "sk-secret", System: "s", Prompt: "p"})
			if !check(err) {
				t.Errorf("%s %d: unexpected %v", p, status, err)
			}

			if err != nil && strings.Contains(err.Error(), "sk-secret") {
				t.Errorf("%s %d: error leaks the key: %v", p, status, err)
			}
		}
	}
}

func TestVerifyOnlyRejectsAnInvalidKey(t *testing.T) {
	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek, llm.Kimi} {
		c, _ := provider(t, 200, `{"data":[],"has_more":false,"first_id":null,"last_id":null}`)
		if err := c.Verify(context.Background(), p, "k"); err != nil {
			t.Errorf("%s: %v", p, err)
		}

		c, _ = provider(t, 401, `{}`)
		if err := c.Verify(context.Background(), p, "k"); !errors.Is(err, llm.ErrRejected) {
			t.Errorf("%s: want rejected, got %v", p, err)
		}

		c, _ = provider(t, 403, `{}`)
		if err := c.Verify(context.Background(), p, "k"); errors.Is(err, llm.ErrRejected) {
			t.Errorf("%s: a restricted key (403) must not count as invalid", p)
		}
	}
}

func TestUnreachableProviderIsTemporary(t *testing.T) {
	c := llm.New(llm.URLs{Anthropic: "http://127.0.0.1:1", OpenAI: "http://127.0.0.1:1", DeepSeek: "http://127.0.0.1:1", Kimi: "http://127.0.0.1:1"})

	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek, llm.Kimi} {
		_, err := c.Complete(context.Background(), llm.Request{Provider: p, Model: "m", Level: "low", Key: "k", System: "s", Prompt: "p"})
		if !errors.Is(err, llm.ErrTemporary) {
			t.Errorf("%s: want temporary, got %v", p, err)
		}
	}
}

func TestValidChoice(t *testing.T) {
	for _, tc := range []struct {
		provider     llm.Provider
		model, level string
		want         bool
	}{
		{llm.Anthropic, "claude-opus-5-5", "max", true},
		{llm.Anthropic, "claude-opus-5-5", "none", false},
		{llm.OpenAI, "gpt-6.1-terra", "none", true},
		{llm.OpenAI, "gpt-6-astra", "none", false},
		{llm.DeepSeek, "deepseek-flash", "off", true},
		{llm.DeepSeek, "deepseek-flash", "xhigh", false},
		{llm.DeepSeek, "claude-opus-5-5", "high", false},
		{llm.Kimi, "kimi-k3.1-pro", "max", true},
		{llm.Kimi, "kimi-k3", "low", true},
		{llm.Kimi, "kimi-k3.1-pro", "off", false}, // K3 always reasons
		{llm.Kimi, "kimi-k3.1-pro", "medium", false},
		{llm.Kimi, "deepseek-flash", "high", false},
		{"nobody", "x", "low", false},
	} {
		if got := llm.Valid(tc.provider, tc.model, tc.level); got != tc.want {
			t.Errorf("%s/%s/%s: got %v want %v", tc.provider, tc.model, tc.level, got, tc.want)
		}
	}
}
