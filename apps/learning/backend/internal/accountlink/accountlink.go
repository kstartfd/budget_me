// Package accountlink ties this app's users to the shared accounts: which
// account a user belongs to, the user an account's owner gets here, and the copy
// of the profile this app keeps.
package accountlink

import (
	"context"
	"errors"

	"github.com/google/uuid"

	ident "budget_me/account"
	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// ErrMismatch means two of this app's users claim the same person; it needs a
// human, so it is never papered over.
var ErrMismatch = apperr.NewInternal("account link mismatch")

// errUserGone is a user deleted (a purge, a test cleaning up) between the moment
// it was listed and the moment it was linked; a backfill just moves on.
var errUserGone = errors.New("accountlink: user gone")

type Links struct {
	q        *db.Queries
	accounts *ident.Store
	clock    clock.Clock
}

func New(q *db.Queries, accounts *ident.Store, clk clock.Clock) Links {
	return Links{q: q, accounts: accounts, clock: clk}
}

// AccountOf returns the account behind a user. A user that predates accounts is
// adopted into the account database on the way, so sessions issued before the
// switch keep working.
func (l Links) AccountOf(ctx context.Context, u db.User) (ident.Account, error) {
	if u.AccountID.Valid {
		return l.accounts.ByID(ctx, ids.FromPg(u.AccountID))
	}

	acc, err := l.accounts.Adopt(ctx, ident.Existing{
		TelegramID:   db.Int8Ptr(u.TelegramID),
		Email:        db.TextPtr(u.Email),
		PasswordHash: db.TextPtr(u.PasswordHash),
		FirstName:    u.FirstName,
		Username:     db.TextPtr(u.Username),
		LanguageCode: db.TextPtr(u.LanguageCode),
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	})
	if err != nil {
		return ident.Account{}, err
	}

	if _, err := l.q.SetUserAccount(ctx, db.SetUserAccountParams{
		ID:        u.ID,
		AccountID: ids.ToPg(acc.ID),
		UpdatedAt: db.Time(l.clock.Now()),
	}); err != nil {
		if db.IsNotFound(err) {
			return ident.Account{}, errUserGone
		}

		if db.IsUniqueViolation(err) {
			return ident.Account{}, ErrMismatch
		}

		return ident.Account{}, err
	}

	if err := l.accounts.Register(ctx, acc.ID, ident.Learning); err != nil {
		return ident.Account{}, err
	}

	return acc, nil
}

// UserFor returns this app's user for an account, creating it on the person's
// first visit here. legacy is a user from before accounts that shares the
// person's Telegram id: it becomes the account's user instead of a second one.
func (l Links) UserFor(ctx context.Context, acc ident.Account, legacy *db.User) (db.User, error) {
	u, err := l.q.GetUserByAccountID(ctx, ids.ToPg(acc.ID))

	switch {
	case err == nil:
	case !db.IsNotFound(err):
		return db.User{}, err
	case legacy != nil && !legacy.AccountID.Valid:
		u, err = l.q.SetUserAccount(ctx, db.SetUserAccountParams{
			ID:        legacy.ID,
			AccountID: ids.ToPg(acc.ID),
			UpdatedAt: db.Time(l.clock.Now()),
		})
		if err != nil {
			return db.User{}, l.classify(err)
		}
	default:
		now := l.clock.Now()

		u, err = l.q.InsertUser(ctx, db.InsertUserParams{
			ID:           ids.ToPg(ids.New()),
			AccountID:    ids.ToPg(acc.ID),
			TelegramID:   db.Int8(acc.TelegramID),
			Email:        db.Text(acc.Email),
			PasswordHash: db.Text(acc.PasswordHash),
			FirstName:    acc.FirstName,
			Username:     db.Text(acc.Username),
			LanguageCode: db.Text(acc.LanguageCode),
			CreatedAt:    db.Time(now),
			UpdatedAt:    db.Time(now),
		})
		if err != nil {
			return db.User{}, l.classify(err)
		}
	}

	if err := l.accounts.Register(ctx, acc.ID, ident.Learning); err != nil {
		return db.User{}, err
	}

	return l.Mirror(ctx, ids.FromPg(u.ID), acc)
}

// Mirror copies the account's identity into this app's user row.
func (l Links) Mirror(ctx context.Context, userID uuid.UUID, acc ident.Account) (db.User, error) {
	u, err := l.q.MirrorAccountToUser(ctx, db.MirrorAccountToUserParams{
		ID:           ids.ToPg(userID),
		TelegramID:   db.Int8(acc.TelegramID),
		Email:        db.Text(acc.Email),
		PasswordHash: db.Text(acc.PasswordHash),
		FirstName:    acc.FirstName,
		Username:     db.Text(acc.Username),
		LanguageCode: db.Text(acc.LanguageCode),
		UpdatedAt:    db.Time(l.clock.Now()),
	})

	return u, l.classify(err)
}

// Backfill adopts every user that predates accounts; it reports how many it
// linked. Safe to run any number of times.
func (l Links) Backfill(ctx context.Context) (int, error) {
	users, err := l.q.ListUsersWithoutAccount(ctx)
	if err != nil {
		return 0, err
	}

	linked := 0

	for _, u := range users {
		_, err := l.AccountOf(ctx, u)

		switch {
		case err == nil:
			linked++
		case errors.Is(err, errUserGone):
		default:
			return 0, err
		}
	}

	return linked, nil
}

func (l Links) classify(err error) error {
	if db.IsUniqueViolation(err) {
		return ErrMismatch
	}

	return err
}
