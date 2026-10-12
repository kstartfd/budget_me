package commands

import "budget_me/backend/internal/apperr"

var (
	ErrLimitNotPositive = apperr.NewInvalid("limit must be positive")
	ErrInvalidCategory  = apperr.NewInvalid("category not found")
	ErrNotExpense       = apperr.NewInvalid("limits apply to expense categories only")
)
