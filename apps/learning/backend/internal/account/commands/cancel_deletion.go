package commands

import (
	"context"

	"github.com/google/uuid"

	ident "budget_me/account"
	"learning_app/backend/internal/accountlink"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
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

	return h.accounts.CancelDeletion(ctx, acc.ID, h.clock.Now())
}
