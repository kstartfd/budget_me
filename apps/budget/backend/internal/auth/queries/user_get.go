package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UserGet struct {
	ID uuid.UUID
}

type GetHandler struct {
	q *db.Queries
}

func NewGetHandler(q *db.Queries) GetHandler {
	return GetHandler{q: q}
}

func (h GetHandler) Handle(ctx context.Context, query UserGet) (db.User, error) {
	return h.q.GetUserByID(ctx, ids.ToPg(query.ID))
}
