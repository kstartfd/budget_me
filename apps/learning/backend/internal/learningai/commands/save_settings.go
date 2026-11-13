package commands

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
)

const verifyTimeout = 10 * time.Second

type SaveSettings struct {
	UserID   uuid.UUID
	Provider llm.Provider
	Model    string
	Level    string
	APIKey   string
}

type SaveHandler struct {
	q     *db.Queries
	clock clock.Clock
	vault secretbox.Box
	llm   llm.Client
}

func NewSaveHandler(q *db.Queries, clk clock.Clock, vault secretbox.Box, client llm.Client) SaveHandler {
	return SaveHandler{q: q, clock: clk, vault: vault, llm: client}
}

// Handle stores the agent choice. An empty APIKey keeps the stored key, which
// is only possible while the provider stays the same: a key belongs to one.
func (h SaveHandler) Handle(ctx context.Context, cmd SaveSettings) (db.LearningAiSetting, error) {
	if !llm.Valid(cmd.Provider, cmd.Model, cmd.Level) {
		return db.LearningAiSetting{}, apperr.NewInvalid("unknown agent or level")
	}

	now := db.Time(h.clock.Now())
	userID := ids.ToPg(cmd.UserID)
	provider := db.LearningAiProvider(cmd.Provider)
	level := db.LearningAiLevel(cmd.Level)

	key := strings.TrimSpace(cmd.APIKey)
	if key == "" {
		row, err := h.q.UpdateLearningAIChoice(ctx, db.UpdateLearningAIChoiceParams{UserID: userID, Provider: provider, Model: cmd.Model, Level: level, UpdatedAt: now})
		if db.IsNotFound(err) {
			return row, apperr.NewInvalid("API key is required")
		}

		return row, err
	}

	if n := utf8.RuneCountInString(key); n < 8 || n > 500 {
		return db.LearningAiSetting{}, apperr.NewInvalid("API key must be 8 to 500 characters")
	}

	if err := h.verify(ctx, cmd.Provider, key); err != nil {
		return db.LearningAiSetting{}, err
	}

	sealed, err := h.vault.Seal([]byte(key), cmd.UserID[:])
	if err != nil {
		return db.LearningAiSetting{}, err
	}

	hint := []rune(key)

	return h.q.UpsertLearningAISettings(ctx, db.UpsertLearningAISettingsParams{UserID: userID, Provider: provider, Model: cmd.Model, Level: level, KeyCiphertext: sealed, KeyHint: string(hint[len(hint)-4:]), UpdatedAt: now})
}

// verify turns the provider's "invalid key" into a message for the form. Any
// other failure (outage, a key restricted from listing models) says nothing
// about the key, so it must not stop the user from saving.
func (h SaveHandler) verify(ctx context.Context, provider llm.Provider, key string) error {
	ctx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()

	err := h.llm.Verify(ctx, provider, key)
	if errors.Is(err, llm.ErrRejected) {
		return apperr.NewInvalid("the provider rejected this API key")
	}

	if err != nil {
		slog.Warn("learning ai key could not be verified", "provider", provider, "err", err)
	}

	return nil
}

// Delete forgets the agent and its key. Nothing to forget is not an error.
func (h SaveHandler) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := h.q.DeleteLearningAISettings(ctx, ids.ToPg(userID))

	return err
}
