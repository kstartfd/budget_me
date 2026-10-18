package commands

import (
	"context"

	ident "budget_me/account"
	"github.com/google/uuid"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type CancelDeletion struct {
	UserID uuid.UUID
}

type CancelDeletionHandler struct {
	q        *db.Queries
	accounts *ident.Store
	links    accountlink.Links
	clock    clock.Clock
}

func NewCancelDeletionHandler(q *db.Queries, accounts *ident.Store, links accountlink.Links, clk clock.Clock) CancelDeletionHandler {
	return CancelDeletionHandler{q: q, accounts: accounts, links: links, clock: clk}
}

// Handle calls the deletion off. Idempotent: cancelling when nothing is pending
// simply clears a NULL.
func (h CancelDeletionHandler) Handle(ctx context.Context, cmd CancelDeletion) (ident.Account, error) {
	acc, err := accountOf(ctx, h.q, h.links, cmd.UserID)
	if err != nil {
		return ident.Account{}, err
	}

	acc, err = h.accounts.CancelDeletion(ctx, acc.ID, h.clock.Now())
	if err != nil {
		return ident.Account{}, err
	}

	_, err = h.q.CancelUserDeletion(ctx, db.CancelUserDeletionParams{
		ID:  ids.ToPg(cmd.UserID),
		Now: db.Time(h.clock.Now()),
	})

	return acc, err
}
