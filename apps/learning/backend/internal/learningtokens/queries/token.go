package queries

import (
	"context"
	"crypto/sha256"
	"log/slog"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/secretbox"
)

type TokenHandler struct {
	q     *db.Queries
	clock clock.Clock
	vault secretbox.Box
}

func NewTokenHandler(q *db.Queries, clk clock.Clock, vault secretbox.Box) TokenHandler {
	return TokenHandler{q: q, clock: clk, vault: vault}
}
func (h TokenHandler) List(ctx context.Context, userID uuid.UUID) ([]db.LearningToken, error) {
	return h.q.ListLearningTokens(ctx, ids.ToPg(userID))
}

// Reveal decrypts the stored copy of an active token for its owner. Tokens
// issued before the encrypted copy existed have none, and a copy sealed under a
// different AUTH_SECRET can no longer be opened; the hash keeps authenticating
// both, they just cannot be shown again.
func (h TokenHandler) Reveal(ctx context.Context, userID, id uuid.UUID) (string, error) {
	row, err := h.q.GetLearningToken(ctx, db.GetLearningTokenParams{ID: ids.ToPg(id), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return "", apperr.NewNotFound("token not found")
	}

	if err != nil {
		return "", err
	}

	if len(row.TokenCiphertext) == 0 {
		return "", apperr.NewNotFound("token cannot be shown")
	}

	raw, err := h.vault.Open(row.TokenCiphertext, id[:])
	if err != nil {
		slog.Error("learning token cannot be decrypted", "token_id", id, "err", err)

		return "", apperr.NewInternal("token cannot be decrypted")
	}

	return string(raw), nil
}

// Authenticate returns the token's owner, whether it is capture-only, and the
// single topic it is bound to (uuid.Nil for a token issued before topic
// scoping existed, which still has the run of the whole account).
func (h TokenHandler) Authenticate(ctx context.Context, raw string) (uuid.UUID, bool, uuid.UUID, error) {
	if len(raw) != 47 {
		return uuid.Nil, false, uuid.Nil, apperr.NewUnauthorized("invalid token")
	}

	hash := sha256.Sum256([]byte(raw))

	row, err := h.q.AuthenticateLearningToken(ctx, db.AuthenticateLearningTokenParams{TokenHash: hash[:], Now: db.Time(h.clock.Now())})
	if db.IsNotFound(err) {
		return uuid.Nil, false, uuid.Nil, apperr.NewUnauthorized("invalid token")
	}

	topicID := uuid.Nil
	if row.TopicID.Valid {
		topicID = ids.FromPg(row.TopicID)
	}

	return ids.FromPg(row.UserID), row.Scope == db.LearningTokenScopeCapture, topicID, err
}
