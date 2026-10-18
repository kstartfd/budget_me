package commands

import (
	"errors"

	ident "budget_me/account"
	"learning_app/backend/internal/apperr"
)

var (
	ErrInvalidCredentials = apperr.NewUnauthorized("invalid credentials")
	ErrEmailTaken         = apperr.NewConflict("email already registered")
	ErrUnknownAccount     = apperr.NewUnauthorized("unknown account")
)

// fromAccount maps the account package's sentinels onto this app's errors.
func fromAccount(err error) error {
	switch {
	case errors.Is(err, ident.ErrInvalidCredentials):
		return ErrInvalidCredentials
	case errors.Is(err, ident.ErrEmailTaken):
		return ErrEmailTaken
	case errors.Is(err, ident.ErrNotFound):
		return ErrUnknownAccount
	default:
		return err
	}
}
