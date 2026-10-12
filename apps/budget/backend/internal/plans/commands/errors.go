package commands

import "budget_me/backend/internal/apperr"

var (
	ErrEmptyTitle        = apperr.NewInvalid("title must not be empty")
	ErrAmountNotPositive = apperr.NewInvalid("amount must be positive")
	ErrInvalidCategory   = apperr.NewInvalid("category not found")
	ErrNotExpense        = apperr.NewInvalid("plans are for expense categories only")
	ErrNotFound          = apperr.NewNotFound("plan not found")
)
