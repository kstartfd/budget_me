package commands

import (
	"context"

	ident "budget_me/account"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type AuthTelegram struct {
	InitData string
}

type AuthTelegramHandler struct {
	q        *db.Queries
	accounts *ident.Store
	links    accountlink.Links
	clock    clock.Clock
	bot      string
}

func NewAuthTelegramHandler(q *db.Queries, accounts *ident.Store, links accountlink.Links, clk clock.Clock, botToken string) AuthTelegramHandler {
	return AuthTelegramHandler{q: q, accounts: accounts, links: links, clock: clk, bot: botToken}
}

func (h AuthTelegramHandler) Handle(ctx context.Context, cmd AuthTelegram) (Session, error) {
	now := h.clock.Now()

	tg, err := ident.VerifyInitData(cmd.InitData, h.bot, now)
	if err != nil {
		return Session{}, apperr.NewUnauthorized(err.Error())
	}

	// A user from before accounts existed: its email login moves into the account.
	var (
		legacy *db.User
		carry  *ident.Credentials
	)

	switch existing, err := h.q.GetUserByTelegramID(ctx, db.Int8(&tg.ID)); {
	case err == nil:
		legacy = &existing

		if !existing.AccountID.Valid {
			carry = &ident.Credentials{Email: db.TextPtr(existing.Email), PasswordHash: db.TextPtr(existing.PasswordHash)}
		}
	case !db.IsNotFound(err):
		return Session{}, err
	}

	acc, err := h.accounts.SignInTelegram(ctx, tg, carry, now)
	if err != nil {
		return Session{}, fromAccount(err)
	}

	user, err := h.links.UserFor(ctx, acc, legacy)
	if err != nil {
		return Session{}, err
	}

	return Session{UserID: ids.FromPg(user.ID), Account: acc}, nil
}
