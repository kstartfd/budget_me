package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// checkExpenseCategory verifies the category exists, is visible to the user
// (shared preset or their own), and is an expense — plans are expenses only.
func checkExpenseCategory(ctx context.Context, q *db.Queries, userID, categoryID uuid.UUID) error {
	visible, err := q.CategoryVisibleToUser(ctx, db.CategoryVisibleToUserParams{
		CategoryID: ids.ToPg(categoryID),
		UserID:     ids.ToPg(userID),
	})
	if err != nil {
		return err
	}

	if !visible {
		return ErrInvalidCategory
	}

	dir, err := q.CategoryDirection(ctx, ids.ToPg(categoryID))
	if err != nil {
		return err
	}

	if dir != db.TxDirectionExpense {
		return ErrNotExpense
	}

	return nil
}
