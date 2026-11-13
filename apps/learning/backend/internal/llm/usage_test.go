package llm_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"learning_app/backend/internal/llm"
)

type recorded struct {
	mu    sync.Mutex
	calls []llm.Call
}

func (r *recorded) Record(_ context.Context, call llm.Call) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, call)
}

func (r *recorded) only(t *testing.T) llm.Call {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.calls) != 1 {
		t.Fatalf("want exactly one recorded call, got %+v", r.calls)
	}

	return r.calls[0]
}

var learner = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func request(p llm.Provider) llm.Request {
	return llm.Request{Provider: p, Model: "m", Level: "low", Key: "k", System: "s", Prompt: "p", UserID: learner, Purpose: llm.PurposeExercises}
}

// TestEachProviderCountsTheCacheItsOwnWay: Anthropic leaves what the cache
// served out of its input, OpenAI and DeepSeek count it inside the prompt; the
// recorder gets three amounts that never overlap, whichever provider answered.
func TestEachProviderCountsTheCacheItsOwnWay(t *testing.T) {
	for name, tc := range map[string]struct {
		provider llm.Provider
		reply    string
		want     llm.Usage
	}{
		"anthropic": {llm.Anthropic, strings.Replace(claudeReply, `"usage":{"input_tokens":1,"output_tokens":1}`, `"usage":{"input_tokens":30,"cache_creation_input_tokens":10,"cache_read_input_tokens":60,"output_tokens":25}`, 1), llm.Usage{Input: 40, Cached: 60, Output: 25}},
		"openai":    {llm.OpenAI, `{"output":[{"type":"message","content":[{"type":"output_text","text":"hi"}]}],"usage":{"input_tokens":100,"input_tokens_details":{"cached_tokens":40},"output_tokens":300}}`, llm.Usage{Input: 60, Cached: 40, Output: 300}},
		"deepseek":  {llm.DeepSeek, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_cache_hit_tokens":70}}`, llm.Usage{Input: 30, Cached: 70, Output: 20}},
		"kimi":      {llm.Kimi, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":50}}}`, llm.Usage{Input: 50, Cached: 50, Output: 20}},
		"kimi flat": {llm.Kimi, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":25}}`, llm.Usage{Input: 75, Cached: 25, Output: 20}},
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := provider(t, 200, tc.reply)
			rec := &recorded{}

			if _, err := c.WithRecorder(rec).Complete(context.Background(), request(tc.provider)); err != nil {
				t.Fatal(err)
			}

			call := rec.only(t)
			if call.Usage != tc.want || call.UserID != learner || call.Purpose != llm.PurposeExercises || call.Provider != tc.provider || call.Model != "m" {
				t.Fatalf("recorded %+v, want usage %+v", call, tc.want)
			}
		})
	}
}

// TestAnAnswerWithNoTextIsStillCounted: the provider charged for the reply the
// learner never saw, and the retry the agent makes after it is a second call.
func TestAnAnswerWithNoTextIsStillCounted(t *testing.T) {
	c, _ := provider(t, 200, `{"choices":[{"message":{"content":""}}],"usage":{"prompt_tokens":10,"completion_tokens":900}}`)
	rec := &recorded{}

	if _, err := c.WithRecorder(rec).Complete(context.Background(), request(llm.DeepSeek)); err == nil {
		t.Fatal("an empty answer is an error")
	}

	if call := rec.only(t); call.Usage != (llm.Usage{Input: 10, Output: 900}) {
		t.Fatalf("recorded %+v", call)
	}
}

func TestNothingIsRecordedWithoutUsageOrWithoutAUser(t *testing.T) {
	rec := &recorded{}

	// The provider told nothing about usage.
	c, _ := provider(t, 200, `{"choices":[{"message":{"content":"hi"}}]}`)
	if _, err := c.WithRecorder(rec).Complete(context.Background(), request(llm.DeepSeek)); err != nil {
		t.Fatal(err)
	}

	// A call nobody owns (a test or a tool) has nowhere to be counted.
	c, _ = provider(t, 200, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)
	req := request(llm.DeepSeek)
	req.UserID = uuid.Nil

	if _, err := c.WithRecorder(rec).Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	// A failed call used nothing the provider reported.
	c, _ = provider(t, 500, `{}`)
	if _, err := c.WithRecorder(rec).Complete(context.Background(), request(llm.DeepSeek)); err == nil {
		t.Fatal("a 500 is an error")
	}

	if len(rec.calls) != 0 {
		t.Fatalf("nothing should be recorded: %+v", rec.calls)
	}
}

func TestTheStreamedChatIsCountedWithItsCache(t *testing.T) {
	c, _ := provider(t, 200, chatStream)
	rec := &recorded{}

	req := chatRequest()
	req.UserID, req.Purpose = learner, llm.PurposeChat

	if _, err := c.WithRecorder(rec).Chat(context.Background(), req, nil); err != nil {
		t.Fatal(err)
	}

	call := rec.only(t)
	if call.Usage != (llm.Usage{Input: 20, Cached: 100, Output: 7}) || call.Purpose != llm.PurposeChat || call.UserID != learner {
		t.Fatalf("recorded %+v", call)
	}
}

func TestAClientWithoutARecorderJustWorks(t *testing.T) {
	c, _ := provider(t, 200, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)

	if text, err := c.Complete(context.Background(), request(llm.DeepSeek)); err != nil || text != "hi" {
		t.Fatalf("%q %v", text, err)
	}
}
