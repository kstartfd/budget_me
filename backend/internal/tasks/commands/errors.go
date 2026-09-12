package commands

import "budget_me/backend/internal/apperr"

var (
	ErrEmptyTitle  = apperr.NewInvalid("title must not be empty")
	ErrInvalidDate = apperr.NewInvalid("date must be YYYY-MM-DD")
	ErrNotFound    = apperr.NewNotFound("task not found")
)
