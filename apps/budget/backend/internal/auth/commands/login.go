package commands

import (
	"context"

	ident "budget_me/account"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/ids"
)

type Login struct {
	Email    string
	Password string
}

type LoginHandler struct {
	accounts *ident.Store
	links    accountlink.Links
}

func NewLoginHandler(accounts *ident.Store, links accountlink.Links) LoginHandler {
	return LoginHandler{accounts: accounts, links: links}
}

func (h LoginHandler) Handle(ctx context.Context, cmd Login) (Session, error) {
	acc, err := h.accounts.SignInEmail(ctx, cmd.Email, cmd.Password)
	if err != nil {
		return Session{}, fromAccount(err)
	}

	user, err := h.links.UserFor(ctx, acc, nil)
	if err != nil {
		return Session{}, err
	}

	return Session{UserID: ids.FromPg(user.ID), Account: acc}, nil
}
