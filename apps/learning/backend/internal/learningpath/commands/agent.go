package commands

import (
	"context"
	"encoding/json"
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

// pathTimeout is the whole budget for one agent call. Reading a test result or
// writing a set of exercises takes one long answer, so this is the same far
// budget the enrichment calls get.
const pathTimeout = 150 * time.Second

// ErrUnreadable marks an answer the agent gave that survived no validation,
// even after one retry told it exactly what was wrong.
var ErrUnreadable = errors.New("learning path: the agent's answer could not be read")

// Completer is the part of llm.Client a path call needs; tests reach the real
// client against a stub provider, like the rest of the app does.
type Completer interface {
	Complete(ctx context.Context, req llm.Request) (string, error)
}

// Agent runs the learner's own agent on one prompt and decodes the JSON it is
// asked for. A reply that does not parse or validate is retried once with the
// complaint attached; the system prompt stays byte-identical so the provider
// cache is not wasted.
type Agent struct {
	q     *db.Queries
	vault secretbox.Box
	llm   Completer
}

func NewAgent(q *db.Queries, vault secretbox.Box, client Completer) Agent {
	return Agent{q: q, vault: vault, llm: client}
}

// completeJSON runs system+prompt through the learner's agent and fills out
// with the JSON object of the reply. parse checks what unmarshalling cannot
// (shape rules); when it complains, the call is made once more with the
// complaint in the prompt — the system prompt stays untouched.
func (a Agent) completeJSON(ctx context.Context, userID uuid.UUID, system, prompt string, out any, parse func() error) error {
	settings, err := a.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return apperr.NewInvalid("this needs an AI agent: set one up in settings")
	}

	if err != nil {
		return err
	}

	key, err := a.vault.Open(settings.KeyCiphertext, userID[:])
	if err != nil {
		slog.Error("learning ai key cannot be decrypted", "user_id", userID, "err", err)

		return apperr.NewConflict("the saved API key cannot be read; enter it again in settings")
	}

	call := func(p string) (string, error) {
		callCtx, cancel := context.WithTimeout(ctx, pathTimeout)
		defer cancel()

		return a.llm.Complete(callCtx, llm.Request{
			Provider: llm.Provider(settings.Provider),
			Model:    settings.Model,
			Level:    string(settings.Level),
			Key:      string(key),
			System:   system,
			Prompt:   p,
		})
	}

	text, err := call(prompt)
	if err != nil {
		return err
	}

	err = decode(text, out, parse)
	if err == nil {
		return nil
	}

	slog.Warn("learning path agent reply did not read, retrying once", "user_id", userID, "err", err)

	retry := prompt + "\n\nYour previous reply could not be used: " + err.Error() + " Reply with the corrected JSON object only."

	text, err = call(retry)
	if err != nil {
		return err
	}

	return decode(text, out, parse)
}

// decode fills out from the JSON object inside the reply and runs parse over
// it. The error says why the reply is unusable, so the retry can quote it.
func decode(text string, out any, parse func() error) error {
	if err := jsonUnmarshal(text, out); err != nil {
		return err
	}

	if parse == nil {
		return nil
	}

	if err := parse(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadable, err)
	}

	return nil
}

// jsonUnmarshal fills out from the JSON object inside the reply, however it is
// wrapped.
func jsonUnmarshal(text string, out any) error {
	raw := sliceJSON(text)
	if raw == nil {
		return fmt.Errorf("%w: no JSON object in the reply", ErrUnreadable)
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadable, err)
	}

	return nil
}
