package commands

import "learning_app/backend/internal/apperr"

var (
	ErrInvalidCredentials = apperr.NewUnauthorized("invalid credentials")
	ErrEmailTaken         = apperr.NewConflict("email already registered")
)
