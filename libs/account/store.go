package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/account/internal/db"
)

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: db.New(pool)}
}

func (s *Store) ByID(ctx context.Context, id uuid.UUID) (Account, error) {
	a, err := s.q.GetAccountByID(ctx, pgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	} else if err != nil {
		return Account{}, err
	}

	return toAccount(a), nil
}

// Credentials is the email login of a user an app already had before accounts
// existed. It is carried into the account the first time that person signs in.
type Credentials struct {
	Email        *string
	PasswordHash *string
}

// SignInTelegram finds the account behind a verified Telegram user, creating it
// on the first sign-in anywhere, and refreshes the profile Telegram reports.
func (s *Store) SignInTelegram(ctx context.Context, tg TelegramUser, carry *Credentials, now time.Time) (Account, error) {
	name := strings.TrimSpace(tg.FirstName + " " + tg.LastName)
	username := optional(tg.Username)
	language := optional(tg.LanguageCode)

	// Two first sign-ins at once race on the unique telegram_id: the loser
	// finds the winner's row on the second pass.
	for range 2 {
		existing, err := s.q.GetAccountByTelegramID(ctx, pgInt8(&tg.ID))

		switch {
		case err == nil:
			updated, err := s.q.UpdateAccountTelegramProfile(ctx, db.UpdateAccountTelegramProfileParams{
				ID:           existing.ID,
				FirstName:    name,
				Username:     pgText(username),
				LanguageCode: pgText(language),
				UpdatedAt:    pgTime(now),
			})
			if err != nil {
				return Account{}, err
			}

			return toAccount(updated), nil
		case !errors.Is(err, pgx.ErrNoRows):
			return Account{}, err
		}

		params := db.InsertAccountParams{
			ID:           pgUUID(uuid.Must(uuid.NewV7())),
			TelegramID:   pgInt8(&tg.ID),
			FirstName:    name,
			Username:     pgText(username),
			LanguageCode: pgText(language),
			CreatedAt:    pgTime(now),
			UpdatedAt:    pgTime(now),
		}

		if carry != nil {
			params.Email = pgText(carry.Email)
			params.PasswordHash = pgText(carry.PasswordHash)
		}

		created, err := s.q.InsertAccount(ctx, params)

		switch constraint(err) {
		case "":
			if err != nil {
				return Account{}, err
			}

			return toAccount(created), nil
		case "accounts_email_key":
			return Account{}, ErrEmailTaken
		}
	}

	return Account{}, errors.New("account: telegram sign-in kept colliding")
}

// SignInEmail checks an email + password and returns the account they belong to.
func (s *Store) SignInEmail(ctx context.Context, email, password string) (Account, error) {
	address := normalizeEmail(email)

	a, err := s.q.GetAccountByEmail(ctx, pgText(&address))
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrInvalidCredentials
	} else if err != nil {
		return Account{}, err
	}

	if !a.PasswordHash.Valid || !VerifyPassword(a.PasswordHash.String, password) {
		return Account{}, ErrInvalidCredentials
	}

	return toAccount(a), nil
}

// LinkEmail adds (or replaces) the email + password login of an account.
func (s *Store) LinkEmail(ctx context.Context, id uuid.UUID, email, password string, now time.Time) (Account, error) {
	address := normalizeEmail(email)

	existing, err := s.q.GetAccountByEmail(ctx, pgText(&address))

	switch {
	case err == nil:
		if uuid.UUID(existing.ID.Bytes) != id {
			return Account{}, ErrEmailTaken
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return Account{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return Account{}, err
	}

	updated, err := s.q.SetAccountWebIdentity(ctx, db.SetAccountWebIdentityParams{
		ID:           pgUUID(id),
		Email:        pgText(&address),
		PasswordHash: pgText(&hash),
		UpdatedAt:    pgTime(now),
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Account{}, ErrNotFound
	case constraint(err) != "":
		return Account{}, ErrEmailTaken
	case err != nil:
		return Account{}, err
	}

	return toAccount(updated), nil
}

// Register records that an app holds data for the account, so a deletion waits
// for that app to let go of it.
func (s *Store) Register(ctx context.Context, id uuid.UUID, app App) error {
	return s.q.RegisterAccountApp(ctx, db.RegisterAccountAppParams{
		AccountID: pgUUID(id),
		App:       db.AppName(app),
	})
}

// Adopt finds the account of a user an app already had before accounts existed,
// matching on Telegram id and then on email, and creates it when there is none.
// Both apps call it for their old users, so a person who has used both ends up
// with one account.
func (s *Store) Adopt(ctx context.Context, in Existing) (Account, error) {
	// A second adopter racing us makes the insert collide; the next pass finds
	// the account it created.
	for range 2 {
		if in.TelegramID != nil {
			a, err := s.q.GetAccountByTelegramID(ctx, pgInt8(in.TelegramID))

			switch {
			case err == nil:
				return toAccount(a), nil
			case !errors.Is(err, pgx.ErrNoRows):
				return Account{}, err
			}
		}

		if in.Email != nil {
			address := normalizeEmail(*in.Email)

			a, err := s.q.GetAccountByEmail(ctx, pgText(&address))

			switch {
			case err == nil:
				return toAccount(a), nil
			case !errors.Is(err, pgx.ErrNoRows):
				return Account{}, err
			}
		}

		created, err := s.q.InsertAccount(ctx, db.InsertAccountParams{
			ID:           pgUUID(uuid.Must(uuid.NewV7())),
			TelegramID:   pgInt8(in.TelegramID),
			Email:        pgText(in.Email),
			PasswordHash: pgText(in.PasswordHash),
			FirstName:    in.FirstName,
			Username:     pgText(in.Username),
			LanguageCode: pgText(in.LanguageCode),
			CreatedAt:    pgTime(in.CreatedAt),
			UpdatedAt:    pgTime(in.UpdatedAt),
		})

		if constraint(err) == "" {
			if err != nil {
				return Account{}, err
			}

			return toAccount(created), nil
		}
	}

	return Account{}, errors.New("account: adopt kept colliding")
}

// Existing is a row of an app's users table from before accounts existed.
type Existing struct {
	TelegramID   *int64
	Email        *string
	PasswordHash *string
	FirstName    string
	Username     *string
	LanguageCode *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func optional(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
