package commands

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	ident "budget_me/account"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// purgeEvery is how often the sweep looks for accounts whose grace period is
// over. A deletion is measured in days, so hours are plenty.
const purgeEvery = 6 * time.Hour

type PurgeHandler struct {
	q        *db.Queries
	accounts *ident.Store
	clock    clock.Clock
	grace    time.Duration
}

func NewPurgeHandler(q *db.Queries, accounts *ident.Store, clk clock.Clock, grace time.Duration) PurgeHandler {
	return PurgeHandler{q: q, accounts: accounts, clock: clk, grace: grace}
}

// Handle erases this app's data of every account whose deletion is due. Safe to
// repeat, and safe to run in two containers at once (blue and green both do).
func (h PurgeHandler) Handle(ctx context.Context) (int, error) {
	cutoff := h.clock.Now().Add(-h.grace)

	purged, err := h.accounts.Purge(ctx, ident.Learning, cutoff, func(ctx context.Context, accountIDs []uuid.UUID) error {
		pg := make([]pgtype.UUID, 0, len(accountIDs))
		for _, id := range accountIDs {
			pg = append(pg, ids.ToPg(id))
		}

		return h.q.DeleteUsersByAccount(ctx, pg)
	})

	return len(purged), err
}

// Run sweeps once at start and then every few hours until ctx ends.
func (h PurgeHandler) Run(ctx context.Context) {
	ticker := time.NewTicker(purgeEvery)
	defer ticker.Stop()

	for {
		if n, err := h.Handle(ctx); err != nil {
			slog.Error("account purge", "err", err)
		} else if n > 0 {
			slog.Info("account purge", "accounts", n)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
