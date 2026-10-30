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

	return llm.New(llm.URLs{Anthropic: srv.URL, OpenAI: srv.URL, DeepSeek: srv.URL}), got
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

func TestProviderStatusesAreSortedForCallers(t *testing.T) {
	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek} {
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
	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek} {
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
	c := llm.New(llm.URLs{Anthropic: "http://127.0.0.1:1", OpenAI: "http://127.0.0.1:1", DeepSeek: "http://127.0.0.1:1"})

	for _, p := range []llm.Provider{llm.Anthropic, llm.OpenAI, llm.DeepSeek} {
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
		{"nobody", "x", "low", false},
	} {
		if got := llm.Valid(tc.provider, tc.model, tc.level); got != tc.want {
			t.Errorf("%s/%s/%s: got %v want %v", tc.provider, tc.model, tc.level, got, tc.want)
		}
	}
}

func TestReaderPicksAModelThatSeesAtALowEffort(t *testing.T) {
	for _, tc := range []struct {
		provider     llm.Provider
		model        string
		wantModel    string
		wantLevel    string
		wantReadable bool
	}{
		{llm.Anthropic, "claude-haiku-5-5", "claude-haiku-5-5", "low", true},
		{llm.OpenAI, "gpt-6-luna", "gpt-6-luna", "low", true},
		{llm.DeepSeek, "deepseek-flash", "deepseek-flash", "off", true},
		// The chosen model cannot see: the provider's own that can stands in.
		{llm.DeepSeek, "deepseek-v4-pro", "deepseek-flash", "off", true},
		// An unknown model of a known provider is not trusted to see either.
		{llm.Anthropic, "claude-unknown", "claude-opus-5-5", "low", true},
		{llm.Provider("nobody"), "x", "", "", false},
	} {
		model, level, ok := llm.Reader(tc.provider, tc.model)
		if model != tc.wantModel || level != tc.wantLevel || ok != tc.wantReadable {
			t.Errorf("%s/%s: got %q %q %v, want %q %q %v", tc.provider, tc.model, model, level, ok, tc.wantModel, tc.wantLevel, tc.wantReadable)
		}

		if ok && !llm.Valid(tc.provider, model, level) {
			t.Errorf("%s/%s: %s at %s is not a choice the catalog offers", tc.provider, tc.model, model, level)
		}
	}
}
