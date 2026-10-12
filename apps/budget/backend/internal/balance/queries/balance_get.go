package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type GetBalance struct {
	UserID uuid.UUID
}

// Balance is money on hand. Set is false until the user has stated an opening
// figure — the app asks for one rather than showing a made-up zero, since
// transactions alone cannot know what was already in the wallet.
type Balance struct {
	Set          bool
	BalanceMinor int64
}

type GetHandler struct {
	q *db.Queries
}

func NewGetHandler(q *db.Queries) GetHandler {
	return GetHandler{q: q}
}

// Handle returns the anchor plus everything that moved since it: income adds,
// expense subtracts. Transactions older than the anchor are ignored on purpose
// — the anchor IS the user's correction, so anything before it is already
// accounted for in the figure they stated.
func (h GetHandler) Handle(ctx context.Context, query GetBalance) (Balance, error) {
	user, err := h.q.GetUserByID(ctx, ids.ToPg(query.UserID))
	if err != nil {
		return Balance{}, err
	}

	if !user.BalanceAnchorMinor.Valid || !user.BalanceAnchorAt.Valid {
		return Balance{}, nil
	}

	net, err := h.q.SumSignedSince(ctx, db.SumSignedSinceParams{
		UserID: ids.ToPg(query.UserID),
		FromAt: user.BalanceAnchorAt,
	})
	if err != nil {
		return Balance{}, err
	}

	return Balance{
		Set:          true,
		BalanceMinor: user.BalanceAnchorMinor.Int64 + net,
	}, nil
}
