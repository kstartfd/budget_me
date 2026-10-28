package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
)

// completeTimeout is the whole budget for one call. A high effort on a large
// model thinks for a while, so this is far longer than the capture queue's 40
// seconds, which a user is not waiting on.
const completeTimeout = 150 * time.Second

// ErrAgentFailed marks an answer the agent could not give for a reason that is
// neither a rejected key nor a temporary outage: a refusal, an empty reply, an
// unreadable response.
var ErrAgentFailed = errors.New("learning ai: the agent did not answer")

// Completer is the part of llm.Client a completion needs; tests swap in a fake.
type Completer interface {
	Complete(ctx context.Context, req llm.Request) (string, error)
}

type Complete struct {
	UserID uuid.UUID
	System string
	Prompt string
}

type CompleteHandler struct {
	q     *db.Queries
	vault secretbox.Box
	llm   Completer
}

func NewCompleteHandler(q *db.Queries, vault secretbox.Box, client Completer) CompleteHandler {
	return CompleteHandler{q: q, vault: vault, llm: client}
}

// Handle runs one prompt through the agent the user chose and returns its
// text. The sealed key is opened here and goes straight to the provider; it is
// never part of the result, and neither it nor the prompt is logged.
func (h CompleteHandler) Handle(ctx context.Context, cmd Complete) (string, error) {
	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(cmd.UserID))
	if db.IsNotFound(err) {
		return "", apperr.NewNotFound("no AI agent chosen")
	}

	if err != nil {
		return "", err
	}

	key, err := h.vault.Open(settings.KeyCiphertext, cmd.UserID[:])
	if err != nil {
		slog.Error("learning ai key cannot be decrypted", "user_id", cmd.UserID, "err", err)

		return "", apperr.NewConflict("the stored API key cannot be read; enter it again")
	}

	ctx, cancel := context.WithTimeout(ctx, completeTimeout)
	defer cancel()

	text, err := h.llm.Complete(ctx, llm.Request{
		Provider: llm.Provider(settings.Provider),
		Model:    settings.Model,
		Level:    string(settings.Level),
		Key:      string(key),
		System:   cmd.System,
		Prompt:   cmd.Prompt,
	})
	if err != nil {
		slog.Warn("learning ai completion failed", "user_id", cmd.UserID, "provider", settings.Provider, "err", err)

		if errors.Is(err, llm.ErrRejected) || errors.Is(err, llm.ErrTemporary) {
			return "", err
		}

		return "", fmt.Errorf("%w: %w", ErrAgentFailed, err)
	}

	return text, nil
}
