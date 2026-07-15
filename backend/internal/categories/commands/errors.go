package commands

import "budget_me/backend/internal/apperr"

var (
	ErrEmptyName = apperr.NewInvalid("name must not be empty")
	ErrNameTaken = apperr.NewConflict("a category with this name already exists")
	ErrNotFound  = apperr.NewNotFound("category not found")
)
