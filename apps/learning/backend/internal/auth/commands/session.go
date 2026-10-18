package commands

import (
	"github.com/google/uuid"

	ident "budget_me/account"
)

// Session is who just signed in: this app's user id, which the token carries,
// and the account behind it.
type Session struct {
	UserID  uuid.UUID
	Account ident.Account
}
