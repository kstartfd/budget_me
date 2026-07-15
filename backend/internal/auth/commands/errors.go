package commands

import "budget_me/backend/internal/apperr"

var (
	ErrInvalidCredentials = apperr.NewUnauthorized("invalid credentials")
	ErrEmailTaken         = apperr.NewConflict("email already registered")
)
