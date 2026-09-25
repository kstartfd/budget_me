package dayemail

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/testdb"
)

func TestConfirmationRequiresCurrentEmailAndOwner(t *testing.T) {
	pool, q := testdb.Pool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	owner := testdb.NewUser(t, pool, q, now, 1)
	other := testdb.NewUser(t, pool, q, now, 1)
	email := "day-email-test@example.com"
	_, err := q.SetUserWebIdentity(ctx, db.SetUserWebIdentityParams{
		ID: ids.ToPg(owner), Email: db.Text(&email),
		PasswordHash: db.Text(&email), UpdatedAt: db.Time(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	hash := "test-confirmation-hash"
	_, err = q.UpsertPendingDayEmailSubscription(ctx, db.UpsertPendingDayEmailSubscriptionParams{
		UserID: ids.ToPg(owner), Email: email, Frequency: db.DayEmailFrequencyTwice,
		ConfirmationHash: db.Text(&hash), ConfirmationExpiresAt: db.Time(now.Add(time.Hour)),
		Now: db.Time(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []db.ConfirmDayEmailSubscriptionParams{
		{UserID: ids.ToPg(other), ConfirmationHash: db.Text(&hash), Now: db.Time(now)},
		{UserID: ids.ToPg(owner), ConfirmationHash: db.Text(&hash), Now: db.Time(now.Add(2 * time.Hour))},
	} {
		if _, err := q.ConfirmDayEmailSubscription(ctx, attempt); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("confirmation with wrong owner or expired token = %v", err)
		}
	}
	changedEmail := "changed-day-email@example.com"
	_, err = q.SetUserWebIdentity(ctx, db.SetUserWebIdentityParams{
		ID: ids.ToPg(owner), Email: db.Text(&changedEmail),
		PasswordHash: db.Text(&changedEmail), UpdatedAt: db.Time(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.ConfirmDayEmailSubscription(ctx, db.ConfirmDayEmailSubscriptionParams{
		UserID: ids.ToPg(owner), ConfirmationHash: db.Text(&hash), Now: db.Time(now),
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("confirmation after email change = %v", err)
	}
	_, err = q.SetUserWebIdentity(ctx, db.SetUserWebIdentityParams{
		ID: ids.ToPg(owner), Email: db.Text(&email),
		PasswordHash: db.Text(&email), UpdatedAt: db.Time(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	confirmed, err := q.ConfirmDayEmailSubscription(ctx, db.ConfirmDayEmailSubscriptionParams{
		UserID: ids.ToPg(owner), ConfirmationHash: db.Text(&hash), Now: db.Time(now),
	})
	if err != nil || !confirmed.ConfirmedAt.Valid {
		t.Fatalf("confirmation = %+v, error = %v", confirmed, err)
	}
	if _, err := q.ConfirmDayEmailSubscription(ctx, db.ConfirmDayEmailSubscriptionParams{
		UserID: ids.ToPg(owner), ConfirmationHash: db.Text(&hash), Now: db.Time(now),
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("reused token = %v", err)
	}
}

func TestDeliverySnapshotIsUniquePerDayAndSlot(t *testing.T) {
	pool, q := testdb.Pool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	userID := testdb.NewUser(t, pool, q, now, 1)
	day, _ := db.ParseDate("2026-09-25")
	first := db.InsertDayEmailDeliveryParams{
		UserID: ids.ToPg(userID), LocalDay: day, Slot: db.DayEmailSlotMorning,
		Email: "test@example.com", Subject: "Original", Body: "First task", Now: db.Time(now),
	}
	if rows, err := q.InsertDayEmailDelivery(ctx, first); err != nil || rows != 1 {
		t.Fatalf("insert = %d, %v", rows, err)
	}
	first.Body = "Changed task"
	if rows, err := q.InsertDayEmailDelivery(ctx, first); err != nil || rows != 0 {
		t.Fatalf("duplicate insert = %d, %v", rows, err)
	}
	stored, err := q.GetDayEmailDelivery(ctx, db.GetDayEmailDeliveryParams{
		UserID: ids.ToPg(userID), LocalDay: day, Slot: db.DayEmailSlotMorning,
	})
	if err != nil || stored.Body != "First task" {
		t.Fatalf("stored body = %q, error = %v", stored.Body, err)
	}
}
