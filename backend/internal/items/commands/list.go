package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// checkListVisible verifies the list exists and is visible to the user
// (shared preset or their own) — mirrors plans' checkExpenseCategory.
func checkListVisible(ctx context.Context, q *db.Queries, userID, listID uuid.UUID) error {
	visible, err := q.ListVisibleToUser(ctx, db.ListVisibleToUserParams{
		ListID: ids.ToPg(listID),
		UserID: ids.ToPg(userID),
	})
	if err != nil {
		return err
	}

	if !visible {
		return ErrInvalidList
	}

	return nil
}

func checkRating(rating *int) error {
	if rating != nil && (*rating < 1 || *rating > 10) {
		return ErrInvalidRating
	}

	return nil
}
