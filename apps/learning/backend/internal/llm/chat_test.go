package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"learning_app/backend/internal/llm"
)

const chatStream = `: keep-alive

data: {"choices":[{"delta":{"role":"assistant","content":""}}]}

data: {"choices":[{"delta":{"reasoning_content":"hidden thought"}}]}

data: {"choices":[{"delta":{"content":"It is "}}]}

data: {"choices":[{"delta":{"content":"a greeting."}}]}

data: {"choices":[],"usage":{"prompt_tokens":120,"completion_tokens":7,"prompt_cache_hit_tokens":100}}

data: [DONE]

`

func chatRequest() llm.ChatRequest {
	return llm.ChatRequest{
		Provider: llm.DeepSeek,
		Model:    "deepseek-v4-pro",
		Key:      "sk-deepseek",
		System:   "you are a tutor",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "what is hi?"}, {Role: llm.RoleAssistant, Content: "a word"}, {Role: llm.RoleUser, Content: "and hello?"}},
	}
}

func TestDeepSeekChatStreamsTheAnswerAndCountsTokens(t *testing.T) {
	c, got := provider(t, 200, chatStream)

	var pieces []string

	res, err := c.Chat(context.Background(), chatRequest(), func(s string) { pieces = append(pieces, s) })
	if err != nil {
		t.Fatal(err)
	}

	if res.Text != "It is a greeting." || strings.Join(pieces, "|") != "It is |a greeting." {
		t.Fatalf("text %q pieces %q", res.Text, pieces)
	}

	if res.PromptTokens != 120 || res.CompletionTokens != 7 || res.CacheHitTokens != 100 || res.Context() != 127 {
		t.Fatalf("usage %+v", res)
	}

	if got.path != "/chat/completions" || got.header.Get("Authorization") != "Bearer sk-deepseek" {
		t.Fatalf("path %q auth %q", got.path, got.header.Get("Authorization"))
	}

	if got.body["model"] != "deepseek-v4-pro" || got.body["stream"] != true || got.body["max_tokens"] != float64(1500) {
		t.Fatalf("body %v", got.body)
	}

	if got.body["thinking"].(map[string]any)["type"] != "disabled" || got.body["stream_options"].(map[string]any)["include_usage"] != true {
		t.Fatalf("thinking and usage must be set: %v", got.body)
	}

	// The system prompt first and the history in order: the prefix has to be
	// identical from one turn to the next for the provider's cache to hit.
	var roles []string

	for _, m := range got.body["messages"].([]any) {
		roles = append(roles, m.(map[string]any)["role"].(string))
	}

	if strings.Join(roles, ",") != "system,user,assistant,user" {
		t.Fatalf("roles %v", roles)
	}
}

func TestDeepSeekChatNeverReturnsReasoning(t *testing.T) {
	c, _ := provider(t, 200, chatStream)

	res, err := c.Chat(context.Background(), chatRequest(), nil)
	if err != nil || strings.Contains(res.Text, "hidden") {
		t.Fatalf("text %q err %v", res.Text, err)
	}
}

func TestDeepSeekChatCutStreamIsTemporaryAndKeepsNothing(t *testing.T) {
	c, _ := provider(t, 200, strings.Replace(chatStream, "data: [DONE]\n\n", "", 1))

	res, err := c.Chat(context.Background(), chatRequest(), nil)
	if !errors.Is(err, llm.ErrTemporary) || res.Text != "" {
		t.Fatalf("a half answer must not pass as an answer: %q %v", res.Text, err)
	}
}

func TestDeepSeekChatEmptyAnswerIsAnError(t *testing.T) {
	c, _ := provider(t, 200, "data: {\"choices\":[{\"delta\":{\"content\":\"  \"}}]}\n\ndata: [DONE]\n\n")

	if _, err := c.Chat(context.Background(), chatRequest(), nil); err == nil || errors.Is(err, llm.ErrTemporary) {
		t.Fatalf("want permanent error, got %v", err)
	}
}

func TestChatStatusesAreSortedForCallers(t *testing.T) {
	c, _ := provider(t, 401, `{}`)
	if _, err := c.Chat(context.Background(), chatRequest(), nil); !errors.Is(err, llm.ErrRejected) {
		t.Fatalf("401: %v", err)
	}

	c, _ = provider(t, 429, `{}`)
	if _, err := c.Chat(context.Background(), chatRequest(), nil); !errors.Is(err, llm.ErrTemporary) {
		t.Fatalf("429: %v", err)
	}
}

func TestChatIsOnlyBuiltForProvidersWithAProfile(t *testing.T) {
	c, _ := provider(t, 200, chatStream)

	req := chatRequest()
	req.Provider = llm.Anthropic

	if _, err := c.Chat(context.Background(), req, nil); !errors.Is(err, llm.ErrChatUnsupported) {
		t.Fatalf("anthropic: %v", err)
	}

	if _, ok := llm.ChatProfileOf(llm.OpenAI); ok {
		t.Fatal("openai has no chat profile yet")
	}

	profile, ok := llm.ChatProfileOf(llm.DeepSeek)
	if !ok || !llm.Valid(llm.DeepSeek, profile.SummaryModel, profile.SummaryLevel) {
		t.Fatalf("the summary call must be a model and level the catalog offers: %+v", profile)
	}
}

func TestChatNeedsAMessage(t *testing.T) {
	c, _ := provider(t, 200, chatStream)

	req := chatRequest()
	req.Messages = nil

	if _, err := c.Chat(context.Background(), req, nil); err == nil {
		t.Fatal("an empty turn must be refused")
	}
}
