package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"budget_me/account/internal/db"
)

// RequestDeletion stamps the request. Nothing is removed here: each app erases
// its data once the grace period is up (Purge). Asking twice keeps the first
// deadline, so a double tap cannot push it back.
func (s *Store) RequestDeletion(ctx context.Context, id uuid.UUID, now time.Time) (Account, error) {
	a, err := s.q.RequestAccountDeletion(ctx, db.RequestAccountDeletionParams{ID: pgUUID(id), Now: pgTime(now)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	} else if err != nil {
		return Account{}, err
	}

	return toAccount(a), nil
}

// CancelDeletion calls a pending deletion off. Idempotent.
func (s *Store) CancelDeletion(ctx context.Context, id uuid.UUID, now time.Time) (Account, error) {
	a, err := s.q.CancelAccountDeletion(ctx, db.CancelAccountDeletionParams{ID: pgUUID(id), Now: pgTime(now)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	} else if err != nil {
		return Account{}, err
	}

	return toAccount(a), nil
}

// Purge carries out the deletions whose deadline (cutoff) has passed, for one
// app: it hands the account ids to deleteLocal so the app removes its own rows,
// lets go of them, and drops every account no app holds any more. deleteLocal
// must be idempotent — a crash between its work and the bookkeeping repeats it.
func (s *Store) Purge(
	ctx context.Context,
	app App,
	cutoff time.Time,
	deleteLocal func(ctx context.Context, ids []uuid.UUID) error,
) ([]uuid.UUID, error) {
	rows, err := s.q.ListAccountsDueForPurge(ctx, db.ListAccountsDueForPurgeParams{
		App:    db.AppName(app),
		Cutoff: pgTime(cutoff),
	})
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, uuid.UUID(r.Bytes))
	}

	due := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		due = append(due, pgUUID(id))
	}

	if len(ids) > 0 {
		if err := deleteLocal(ctx, ids); err != nil {
			return nil, err
		}

		if err := s.q.UnregisterAccountApp(ctx, db.UnregisterAccountAppParams{App: db.AppName(app), Ids: due}); err != nil {
			return nil, err
		}
	}

	if _, err := s.q.DeleteReleasedAccounts(ctx, pgTime(cutoff)); err != nil {
		return nil, err
	}

	return ids, nil
}

// Merge folds the loser account into the survivor, for the operator who finds one
// person under two accounts: the survivor inherits every login it lacks, holds
// data for every app the loser did, and the loser row goes. The apps' own data
// is the caller's business.
func (s *Store) Merge(ctx context.Context, survivor, loser uuid.UUID, now time.Time) (Account, error) {
	if survivor == loser {
		return s.ByID(ctx, survivor)
	}

	var merged db.Account

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		l, err := q.GetAccountByID(ctx, pgUUID(loser))
		if err != nil {
			return err
		}

		if err := q.MoveAccountApps(ctx, db.MoveAccountAppsParams{Survivor: pgUUID(survivor), Loser: pgUUID(loser)}); err != nil {
			return err
		}

		// The loser goes first: its telegram id and email are unique.
		if err := q.DeleteAccount(ctx, pgUUID(loser)); err != nil {
			return err
		}

		merged, err = q.InheritAccountIdentity(ctx, db.InheritAccountIdentityParams{
			ID:           pgUUID(survivor),
			TelegramID:   l.TelegramID,
			Email:        l.Email,
			PasswordHash: l.PasswordHash,
			Username:     l.Username,
			LanguageCode: l.LanguageCode,
			Now:          pgTime(now),
		})

		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	} else if err != nil {
		return Account{}, err
	}

	return toAccount(merged), nil
}
