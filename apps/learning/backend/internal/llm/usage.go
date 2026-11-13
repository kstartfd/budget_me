package llm

import (
	"context"

	"github.com/google/uuid"
)

// Usage is what one call to a provider used, in the three amounts providers
// price by. The providers count differently (Anthropic's input leaves the cache
// out, OpenAI's and DeepSeek's include it), so each client turns what it is told
// into these three, which never overlap.
type Usage struct {
	// Input is the prompt read at the full price: what the provider's cache did not serve.
	Input int
	// Cached is the prompt the provider served from its cache, at a fraction of the price.
	Cached int
	// Output is the reply, thinking included: hidden thoughts are billed as reply.
	Output int
}

// Zero reports a call that told nothing about what it used.
func (u Usage) Zero() bool { return u == Usage{} }

// usageOf builds a Usage from a provider that counts the cache inside its prompt.
func usageOf(prompt, cached, completion int) Usage {
	cached = min(max(cached, 0), max(prompt, 0))

	return Usage{Input: max(prompt, 0) - cached, Cached: cached, Output: max(completion, 0)}
}

// What a call was for. The admin groups spending by it, so a new use of the
// agent gets a name here rather than a string made up on the spot.
const (
	PurposeTestQuestions    = "test-questions"
	PurposeTestJudge        = "test-judge"
	PurposeProgram          = "program"
	PurposePlan             = "plan"
	PurposeExercises        = "exercises"
	PurposeReview           = "review"
	PurposePracticeExamples = "practice-examples"
	PurposePracticeDrills   = "practice-drills"
	PurposeChat             = "chat"
	PurposeChatSummary      = "chat-summary"
	PurposeFill             = "fill"
	PurposeEnrich           = "enrich"
	PurposeComplete         = "complete"
)

// Call is one answered call to a provider, as the recorder is told of it.
type Call struct {
	UserID   uuid.UUID
	Purpose  string
	Provider Provider
	Model    string
	Usage    Usage
}

// Recorder is told of every call that used tokens, whether or not its reply
// turned out to be usable: the provider charged for it all the same. It must
// not fail the call, so it returns nothing.
type Recorder interface {
	Record(ctx context.Context, call Call)
}

// WithRecorder returns a client that tells rec of every call it makes.
func (c Client) WithRecorder(rec Recorder) Client {
	c.rec = rec

	return c
}

// record passes a call on. It is written on a context that outlives the
// request: a learner who closed the page mid-reply still used the tokens.
func (c Client) record(ctx context.Context, call Call) {
	if c.rec == nil || call.UserID == uuid.Nil || call.Usage.Zero() {
		return
	}

	c.rec.Record(context.WithoutCancel(ctx), call)
}
