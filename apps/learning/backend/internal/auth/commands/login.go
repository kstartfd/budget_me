package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"learning_app/backend/internal/db"
)

type Login struct {
	Email    string
	Password string
}

type LoginHandler struct {
	q *db.Queries
}

func NewLoginHandler(q *db.Queries) LoginHandler {
	return LoginHandler{q: q}
}

func (h LoginHandler) Handle(ctx context.Context, cmd Login) (db.User, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))

	user, err := h.q.GetUserByEmail(ctx, db.Text(&email))
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrInvalidCredentials
	} else if err != nil {
		return db.User{}, err
	}

	if !user.PasswordHash.Valid || !VerifyPassword(user.PasswordHash.String, cmd.Password) {
		return db.User{}, ErrInvalidCredentials
	}

	return user, nil
}
