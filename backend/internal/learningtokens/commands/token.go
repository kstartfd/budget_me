package commands

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CreateToken struct {
	UserID uuid.UUID
	Name   string
	Scope  db.LearningTokenScope
}
type TokenHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTokenHandler(q *db.Queries, clk clock.Clock) TokenHandler {
	return TokenHandler{q: q, clock: clk}
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

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return db.LearningToken{}, "", err
	}

	raw := "bml_" + base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(raw))
	row, err := h.q.InsertLearningToken(ctx, db.InsertLearningTokenParams{ID: ids.ToPg(ids.New()), UserID: ids.ToPg(cmd.UserID), Name: name, Prefix: raw[:12], TokenHash: hash[:], Scope: cmd.Scope, CreatedAt: db.Time(h.clock.Now())})

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
