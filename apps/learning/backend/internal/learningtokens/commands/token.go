package commands

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/secretbox"
)

type CreateToken struct {
	UserID  uuid.UUID
	Name    string
	Scope   db.LearningTokenScope
	TopicID uuid.UUID
}
type TokenHandler struct {
	q     *db.Queries
	clock clock.Clock
	vault secretbox.Box
}

func NewTokenHandler(q *db.Queries, clk clock.Clock, vault secretbox.Box) TokenHandler {
	return TokenHandler{q: q, clock: clk, vault: vault}
}

func (h TokenHandler) Handle(ctx context.Context, cmd CreateToken) (db.LearningToken, string, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return db.LearningToken{}, "", apperr.NewInvalid("token name must be 1..80 characters")
	}

	if cmd.Scope == "" {
		cmd.Scope = db.LearningTokenScopeCapture
	}

	if cmd.Scope != db.LearningTokenScopeCapture && cmd.Scope != db.LearningTokenScopeReadWrite {
		return db.LearningToken{}, "", apperr.NewInvalid("invalid token scope")
	}

	if cmd.TopicID == uuid.Nil {
		return db.LearningToken{}, "", apperr.NewInvalid("topicId is required")
	}

	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(cmd.TopicID), UserID: ids.ToPg(cmd.UserID)}); err != nil {
		if db.IsNotFound(err) {
			return db.LearningToken{}, "", apperr.NewNotFound("topic not found")
		}

		return db.LearningToken{}, "", err
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return db.LearningToken{}, "", err
	}

	raw := "bml_" + base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(raw))
	id := ids.New()

	sealed, err := h.vault.Seal([]byte(raw), id[:])
	if err != nil {
		return db.LearningToken{}, "", err
	}

	row, err := h.q.InsertLearningToken(ctx, db.InsertLearningTokenParams{ID: ids.ToPg(id), UserID: ids.ToPg(cmd.UserID), TopicID: ids.ToPg(cmd.TopicID), Name: name, Prefix: raw[:12], TokenHash: hash[:], TokenCiphertext: sealed, Scope: cmd.Scope, CreatedAt: db.Time(h.clock.Now())})

	return row, raw, err
}

func (h TokenHandler) Revoke(ctx context.Context, userID, id uuid.UUID) error {
	n, err := h.q.RevokeLearningToken(ctx, db.RevokeLearningTokenParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID), RevokedAt: db.Time(h.clock.Now())})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewNotFound("token not found")
	}

	return nil
}
