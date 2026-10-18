package commands

import (
	"context"

	ident "budget_me/account"
	"github.com/google/uuid"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type RequestDeletion struct {
	UserID uuid.UUID
}

type RequestDeletionHandler struct {
	q        *db.Queries
	accounts *ident.Store
	links    accountlink.Links
	clock    clock.Clock
}

func NewRequestDeletionHandler(q *db.Queries, accounts *ident.Store, links accountlink.Links, clk clock.Clock) RequestDeletionHandler {
	return RequestDeletionHandler{q: q, accounts: accounts, links: links, clock: clk}
}

// Handle marks the whole account for deletion — every app that holds data for it
// lets go once the grace period is up. Nothing is removed here. Asking twice
// keeps the first deadline (the SQL COALESCEs), so a double tap cannot extend
// the window.
func (h RequestDeletionHandler) Handle(ctx context.Context, cmd RequestDeletion) (ident.Account, error) {
	acc, err := accountOf(ctx, h.q, h.links, cmd.UserID)
	if err != nil {
		return ident.Account{}, err
	}

	acc, err = h.accounts.RequestDeletion(ctx, acc.ID, h.clock.Now())
	if err != nil {
		return ident.Account{}, err
	}

	// The nightly advice pass skips users on this flag, so it stays in step.
	_, err = h.q.RequestUserDeletion(ctx, db.RequestUserDeletionParams{
		ID:  ids.ToPg(cmd.UserID),
		Now: db.Time(h.clock.Now()),
	})

	return acc, err
}

func accountOf(ctx context.Context, q *db.Queries, links accountlink.Links, userID uuid.UUID) (ident.Account, error) {
	user, err := q.GetUserByID(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return ident.Account{}, apperr.NewUnauthorized("unknown account")
	} else if err != nil {
		return ident.Account{}, err
	}

	return links.AccountOf(ctx, user)
}
