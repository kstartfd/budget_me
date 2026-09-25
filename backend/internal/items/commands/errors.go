package commands

import "budget_me/backend/internal/apperr"

var (
	ErrEmptyTitle     = apperr.NewInvalid("title must not be empty")
	ErrInvalidList    = apperr.NewInvalid("list not found")
	ErrInvalidRating  = apperr.NewInvalid("rating must be between 1 and 10")
	ErrInvalidVerdict = apperr.NewInvalid(`verdict must be "works", "meh", "no" or "unknown"`)
	ErrNotFound       = apperr.NewNotFound("item not found")
	ErrInvalidPhoto   = apperr.NewInvalid("unsupported image file")
)
