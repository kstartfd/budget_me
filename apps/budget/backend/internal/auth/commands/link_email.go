package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type LinkEmail struct {
	UserID   uuid.UUID
	Email    string
	Password string
}

type LinkEmailHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewLinkEmailHandler(q *db.Queries, clk clock.Clock) LinkEmailHandler {
	return LinkEmailHandler{q: q, clock: clk}
}

func (h LinkEmailHandler) Handle(ctx context.Context, cmd LinkEmail) (db.User, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))

	existing, err := h.q.GetUserByEmail(ctx, db.Text(&email))
	switch {
	case err == nil:
		if ids.FromPg(existing.ID) != cmd.UserID {
			return db.User{}, ErrEmailTaken
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return db.User{}, err
	}

	hash, err := HashPassword(cmd.Password)
	if err != nil {
		return db.User{}, err
	}

	return h.q.SetUserWebIdentity(ctx, db.SetUserWebIdentityParams{
		ID:           ids.ToPg(cmd.UserID),
		Email:        db.Text(&email),
		PasswordHash: db.Text(&hash),
		UpdatedAt:    db.Time(h.clock.Now()),
	})
}
