package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrChatUnsupported means chat is not built for the user's provider yet.
var ErrChatUnsupported = errors.New("llm: chat is not available for this provider")

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	Role    string
	Content string
}

// ChatRequest is one turn of a conversation. Messages must only ever grow at
// the end between turns: the providers cache the longest identical prefix, so
// an untouched history is read back at a fraction of the price.
type ChatRequest struct {
	Provider Provider
	Model    string
	Key      string
	System   string
	Messages []Message
	// UserID and Purpose say whose turn it is and what for, for the usage record.
	UserID  uuid.UUID
	Purpose string
}

type ChatResult struct {
	Text             string
	PromptTokens     int
	CompletionTokens int
	// CacheHitTokens is how much of the prompt the provider served from cache.
	CacheHitTokens int
}

// Context is what the next turn starts from: this prompt and the reply.
func (r ChatResult) Context() int { return r.PromptTokens + r.CompletionTokens }

// ChatProfile is a provider's own answer to "how do we keep a long chat
// cheap". Each provider gets its own because the caches differ: a rule that
// suits one would defeat another's.
type ChatProfile struct {
	// MaxReply caps one answer.
	MaxReply int
	// CompactAt is the context size, in tokens, at which older turns are folded
	// into the summary. Folding rewrites the start of the prompt and so empties
	// the cache once; a high mark keeps that rare.
	CompactAt int
	// KeepTail is how many of the newest messages stay word for word.
	KeepTail int
	// SummaryModel and SummaryLevel pick the cheap call that writes the summary.
	SummaryModel string
	SummaryLevel string
	// BudgetTokens is what one chat may cost (prompt and reply tokens of all
	// its turns) within BudgetWindow. A long conversation about one phrase
	// helps no one; after the window it starts afresh.
	BudgetTokens int
	BudgetWindow time.Duration
	// UserDayTokens caps what one user's chats cost together per UTC day, and
	// KeyDayTokens what one API key costs across every account that uses it.
	// The user's is keyed by the user, never by the key, so changing keys buys
	// nothing; the key's is the looser of the two so people sharing a key are
	// not the ones to hit it.
	UserDayTokens int
	KeyDayTokens  int
}

var chatProfiles = map[Provider]ChatProfile{
	// DeepSeek caches the repeated prefix of a prompt by itself. Folding at
	// 7k tokens leaves a long stretch of turns that all hit the cache, and the
	// flash model without thinking writes the summary for almost nothing.
	//
	// A turn costs roughly 1k tokens early on and a few thousand once the
	// history has grown, so 120k tokens a day are around 40 questions on one phrase.
	DeepSeek: {
		MaxReply: 1500, CompactAt: 7000, KeepTail: 8, SummaryModel: "deepseek-flash", SummaryLevel: "off",
		BudgetTokens: 120_000, BudgetWindow: 24 * time.Hour,
		UserDayTokens: 400_000, KeyDayTokens: 800_000,
	},
}

// ChatProfileOf reports false for a provider chat is not built for.
func ChatProfileOf(provider Provider) (ChatProfile, bool) {
	profile, ok := chatProfiles[provider]

	return profile, ok
}

// Chat sends one turn and calls onDelta with each piece of the reply as it
// arrives (it may be nil). Only a reply that arrived whole is returned.
func (c Client) Chat(ctx context.Context, req ChatRequest, onDelta func(string)) (ChatResult, error) {
	if len(req.Messages) == 0 {
		return ChatResult{}, errors.New("llm: a chat turn needs a message")
	}

	profile, ok := ChatProfileOf(req.Provider)
	if !ok {
		return ChatResult{}, fmt.Errorf("%w: %s", ErrChatUnsupported, req.Provider)
	}

	switch req.Provider {
	case DeepSeek:
		result, err := c.chatDeepSeek(ctx, req, profile, onDelta)
		if err == nil {
			c.record(ctx, Call{
				UserID: req.UserID, Purpose: req.Purpose, Provider: req.Provider, Model: req.Model,
				Usage: usageOf(result.PromptTokens, result.CacheHitTokens, result.CompletionTokens),
			})
		}

		return result, err
	}

	return ChatResult{}, fmt.Errorf("%w: %s", ErrChatUnsupported, req.Provider)
}
