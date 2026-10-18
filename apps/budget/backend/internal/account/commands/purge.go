package commands

import (
	"context"
	"time"

	ident "budget_me/account"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type PurgeHandler struct {
	q        *db.Queries
	accounts *ident.Store
	clock    clock.Clock
	grace    time.Duration
}

func NewPurgeHandler(q *db.Queries, accounts *ident.Store, clk clock.Clock, grace time.Duration) PurgeHandler {
	return PurgeHandler{q: q, accounts: accounts, clock: clk, grace: grace}
}

// Handle erases this app's data of every account whose deletion is due and
// returns the account ids it let go of. Everything a user owns follows via
// ON DELETE CASCADE. Safe to repeat.
func (h PurgeHandler) Handle(ctx context.Context) ([]uuid.UUID, error) {
	cutoff := h.clock.Now().Add(-h.grace)

	return h.accounts.Purge(ctx, ident.Budget, cutoff, func(ctx context.Context, accountIDs []uuid.UUID) error {
		pg := make([]pgtype.UUID, 0, len(accountIDs))
		for _, id := range accountIDs {
			pg = append(pg, ids.ToPg(id))
		}

		return h.q.DeleteUsersByAccount(ctx, pg)
	})
}
