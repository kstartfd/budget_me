package queries

import (
	"context"
	"crypto/sha256"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type TokenHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTokenHandler(q *db.Queries, clk clock.Clock) TokenHandler {
	return TokenHandler{q: q, clock: clk}
}
func (h TokenHandler) List(ctx context.Context, userID uuid.UUID) ([]db.LearningToken, error) {
	return h.q.ListLearningTokens(ctx, ids.ToPg(userID))
}
func (h TokenHandler) Authenticate(ctx context.Context, raw string) (uuid.UUID, bool, error) {
	if len(raw) != 47 {
		return uuid.Nil, false, apperr.NewUnauthorized("invalid token")
	}

	hash := sha256.Sum256([]byte(raw))

	row, err := h.q.AuthenticateLearningToken(ctx, db.AuthenticateLearningTokenParams{TokenHash: hash[:], Now: db.Time(h.clock.Now())})
	if db.IsNotFound(err) {
		return uuid.Nil, false, apperr.NewUnauthorized("invalid token")
	}

	return ids.FromPg(row.UserID), row.Scope == db.LearningTokenScopeCapture, err
}
