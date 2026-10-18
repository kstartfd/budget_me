package commands

import (
	"context"

	"github.com/google/uuid"

	ident "budget_me/account"
	"learning_app/backend/internal/accountlink"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type LinkEmail struct {
	UserID   uuid.UUID
	Email    string
	Password string
}

type LinkEmailHandler struct {
	q        *db.Queries
	accounts *ident.Store
	links    accountlink.Links
	clock    clock.Clock
}

func NewLinkEmailHandler(q *db.Queries, accounts *ident.Store, links accountlink.Links, clk clock.Clock) LinkEmailHandler {
	return LinkEmailHandler{q: q, accounts: accounts, links: links, clock: clk}
}

func (h LinkEmailHandler) Handle(ctx context.Context, cmd LinkEmail) (ident.Account, error) {
	user, err := h.q.GetUserByID(ctx, ids.ToPg(cmd.UserID))
	if err != nil {
		return ident.Account{}, err
	}

	acc, err := h.links.AccountOf(ctx, user)
	if err != nil {
		return ident.Account{}, fromAccount(err)
	}

	acc, err = h.accounts.LinkEmail(ctx, acc.ID, cmd.Email, cmd.Password, h.clock.Now())
	if err != nil {
		return ident.Account{}, fromAccount(err)
	}

	if _, err := h.links.Mirror(ctx, cmd.UserID, acc); err != nil {
		return ident.Account{}, err
	}

	return acc, nil
}
