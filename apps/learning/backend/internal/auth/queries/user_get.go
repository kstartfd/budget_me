package queries

import (
	"context"

	"github.com/google/uuid"

	ident "budget_me/account"
	"learning_app/backend/internal/accountlink"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type UserGet struct {
	ID uuid.UUID
}

type GetHandler struct {
	q     *db.Queries
	links accountlink.Links
}

func NewGetHandler(q *db.Queries, links accountlink.Links) GetHandler {
	return GetHandler{q: q, links: links}
}

// Handle returns the account of the signed-in user: identity lives there, not
// in this app's copy.
func (h GetHandler) Handle(ctx context.Context, query UserGet) (ident.Account, error) {
	user, err := h.q.GetUserByID(ctx, ids.ToPg(query.ID))
	if err != nil {
		return ident.Account{}, err
	}

	return h.links.AccountOf(ctx, user)
}
