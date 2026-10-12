package queries

import (
	"context"
	"crypto/sha256"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
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
