// Package account is the one place a person's identity lives for every app in
// this repository: the Telegram and email+password logins, the profile they
// carry, and the request to delete the account. An app keeps its own data plus a
// users(id) row whose id equals the account id; it asks this package who a login
// belongs to and which accounts are due to go.
//
// The package returns plain sentinel errors and takes `now` from the caller, so
// it knows nothing about HTTP or about an app's clock.
package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"budget_me/account/internal/db"
)

var (
	ErrNotFound           = errors.New("account not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
)

// App names an app that keeps data for accounts.
type App string

const (
	Budget   App = "budget"
	Learning App = "learning"
)

type Account struct {
	ID                  uuid.UUID
	TelegramID          *int64
	Email               *string
	PasswordHash        *string
	FirstName           string
	Username            *string
	LanguageCode        *string
	DeletionRequestedAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DeleteAt is when a requested deletion will be carried out, nil if none is pending.
func (a Account) DeleteAt(grace time.Duration) *time.Time {
	if a.DeletionRequestedAt == nil {
		return nil
	}

	at := a.DeletionRequestedAt.Add(grace)

	return &at
}

func toAccount(a db.Account) Account {
	acc := Account{
		ID:           uuid.UUID(a.ID.Bytes),
		TelegramID:   int8Ptr(a.TelegramID),
		Email:        textPtr(a.Email),
		PasswordHash: textPtr(a.PasswordHash),
		FirstName:    a.FirstName,
		Username:     textPtr(a.Username),
		LanguageCode: textPtr(a.LanguageCode),
		CreatedAt:    a.CreatedAt.Time,
		UpdatedAt:    a.UpdatedAt.Time,
	}

	if a.DeletionRequestedAt.Valid {
		at := a.DeletionRequestedAt.Time
		acc.DeletionRequestedAt = &at
	}

	return acc
}

func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

func pgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func pgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *s, Valid: true}
}

func pgInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: *i, Valid: true}
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}

	return &t.String
}

func int8Ptr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}

	return &i.Int64
}

// constraint reports which unique constraint an error violated, "" if it is not
// a unique violation.
func constraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}

	return ""
}
