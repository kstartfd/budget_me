package commands

import "budget_me/backend/internal/apperr"

var ErrEmptyName = apperr.NewInvalid("name must not be empty")
