package account_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/account"
)

var now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

// pool connects to ACCOUNT_DATABASE_URL and SKIPS the test when it is unset, so
// `go test ./...` stays green with no database attached.
func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("ACCOUNT_DATABASE_URL")
	if url == "" {
		t.Skip("ACCOUNT_DATABASE_URL not set — skipping DB-backed test")
	}

	p, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(p.Close)

	return p
}

func store(t *testing.T) (*account.Store, *pgxpool.Pool) {
	t.Helper()

	p := pool(t)

	return account.New(p), p
}

// tg returns a Telegram user with an id no other test uses.
func tg(t *testing.T) account.TelegramUser {
	t.Helper()

	return account.TelegramUser{
		ID:        time.Now().UnixNano() % 1_000_000_000_000,
		FirstName: "Ann",
		LastName:  "Lee",
		Username:  "ann",
	}
}

func cleanup(t *testing.T, p *pgxpool.Pool, ids ...uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = p.Exec(context.Background(), "DELETE FROM accounts WHERE id = $1", id)
		}
	})
}

func TestSignInTelegramCreatesThenReusesAndRefreshesProfile(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()
	user := tg(t)

	first, err := s.SignInTelegram(ctx, user, nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, first.ID)

	if first.FirstName != "Ann Lee" || first.Username == nil || *first.Username != "ann" {
		t.Fatalf("profile = %+v", first)
	}

	user.Username = "ann2"

	again, err := s.SignInTelegram(ctx, user, nil, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if again.ID != first.ID {
		t.Fatalf("second sign-in made a new account: %s vs %s", again.ID, first.ID)
	}

	if again.Username == nil || *again.Username != "ann2" {
		t.Fatalf("username not refreshed: %v", again.Username)
	}
}

func TestSignInTelegramCarriesAnExistingEmailLogin(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	email := "known-" + uuid.NewString() + "@example.com"
	hash, _ := account.HashPassword("secret-pass")

	acc, err := s.SignInTelegram(ctx, tg(t), &account.Credentials{Email: &email, PasswordHash: &hash}, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, acc.ID)

	if _, err := s.SignInEmail(ctx, email, "secret-pass"); err != nil {
		t.Fatalf("the carried email login should work: %v", err)
	}
}

func TestSignInEmail(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	acc, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, acc.ID)

	email := "Mixed-" + uuid.NewString() + "@Example.com"

	if _, err := s.LinkEmail(ctx, acc.ID, email, "correct horse", now); err != nil {
		t.Fatal(err)
	}

	got, err := s.SignInEmail(ctx, "  "+email+"  ", "correct horse")
	if err != nil || got.ID != acc.ID {
		t.Fatalf("sign-in with the linked email = %v, %v", got.ID, err)
	}

	if _, err := s.SignInEmail(ctx, email, "wrong"); !errors.Is(err, account.ErrInvalidCredentials) {
		t.Fatalf("wrong password err = %v", err)
	}

	if _, err := s.SignInEmail(ctx, "nobody-"+uuid.NewString()+"@example.com", "x"); !errors.Is(err, account.ErrInvalidCredentials) {
		t.Fatalf("unknown email err = %v", err)
	}
}

func TestLinkEmailRefusesAnotherAccountsAddress(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	a, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	b, err := s.SignInTelegram(ctx, account.TelegramUser{ID: tg(t).ID + 1, FirstName: "Bob"}, nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, a.ID, b.ID)

	email := "taken-" + uuid.NewString() + "@example.com"

	if _, err := s.LinkEmail(ctx, a.ID, email, "password-a", now); err != nil {
		t.Fatal(err)
	}

	if _, err := s.LinkEmail(ctx, b.ID, email, "password-b", now); !errors.Is(err, account.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}

	// Re-linking your own address (changing the password) is fine.
	if _, err := s.LinkEmail(ctx, a.ID, email, "password-a2", now); err != nil {
		t.Fatalf("relink own email: %v", err)
	}
}

func TestDeletionKeepsTheFirstDeadlineAndCanBeCancelled(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	acc, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, acc.ID)

	first, err := s.RequestDeletion(ctx, acc.ID, now)
	if err != nil || first.DeletionRequestedAt == nil {
		t.Fatalf("request = %+v, %v", first, err)
	}

	second, err := s.RequestDeletion(ctx, acc.ID, now.Add(48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if !second.DeletionRequestedAt.Equal(*first.DeletionRequestedAt) {
		t.Fatalf("deadline moved from %v to %v", first.DeletionRequestedAt, second.DeletionRequestedAt)
	}

	if at := second.DeleteAt(30 * 24 * time.Hour); at == nil || !at.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("DeleteAt = %v", at)
	}

	cancelled, err := s.CancelDeletion(ctx, acc.ID, now)
	if err != nil || cancelled.DeletionRequestedAt != nil {
		t.Fatalf("cancel = %+v, %v", cancelled, err)
	}
}

func TestPurgeWaitsForEveryAppThatHoldsTheAccount(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	acc, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, acc.ID)

	for _, app := range []account.App{account.Budget, account.Learning} {
		if err := s.Register(ctx, acc.ID, app); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := s.RequestDeletion(ctx, acc.ID, now); err != nil {
		t.Fatal(err)
	}

	grace := 30 * 24 * time.Hour

	var erased []uuid.UUID

	deleteLocal := func(_ context.Context, ids []uuid.UUID) error {
		erased = append(erased, ids...)

		return nil
	}

	// Inside the grace period nothing is due.
	if got, err := s.Purge(ctx, account.Budget, now.Add(grace-time.Hour).Add(-grace), deleteLocal); err != nil || len(got) != 0 {
		t.Fatalf("early purge = %v, %v", got, err)
	}

	due := now.Add(grace + time.Hour).Add(-grace)

	if got, err := s.Purge(ctx, account.Budget, due, deleteLocal); err != nil || len(got) != 1 {
		t.Fatalf("budget purge = %v, %v", got, err)
	}

	// Learning still holds data: the account row must survive the first purge.
	if _, err := s.ByID(ctx, acc.ID); err != nil {
		t.Fatalf("account vanished while learning still holds it: %v", err)
	}

	if got, err := s.Purge(ctx, account.Learning, due, deleteLocal); err != nil || len(got) != 1 {
		t.Fatalf("learning purge = %v, %v", got, err)
	}

	if _, err := s.ByID(ctx, acc.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("account should be gone once both apps let go: %v", err)
	}

	if len(erased) != 2 {
		t.Fatalf("each app should erase once, erased = %v", erased)
	}
}

func TestAdoptFindsOrCreatesOneAccountPerPerson(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	tgID := tg(t).ID
	email := "adopt-" + uuid.NewString() + "@example.com"
	hash, _ := account.HashPassword("secret-pass")

	// One app's old user: Telegram plus an email login.
	first, err := s.Adopt(ctx, account.Existing{
		TelegramID: &tgID, Email: &email, PasswordHash: &hash,
		FirstName: "Imp", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, first.ID)

	// The other app's old user for the same person, found by Telegram id …
	byTelegram, err := s.Adopt(ctx, account.Existing{TelegramID: &tgID, FirstName: "Imp", CreatedAt: now, UpdatedAt: now})
	if err != nil || byTelegram.ID != first.ID {
		t.Fatalf("by telegram = %v, %v; want %s", byTelegram.ID, err, first.ID)
	}

	// … or, with no Telegram id, by email.
	byEmail, err := s.Adopt(ctx, account.Existing{Email: &email, FirstName: "Imp", CreatedAt: now, UpdatedAt: now})
	if err != nil || byEmail.ID != first.ID {
		t.Fatalf("by email = %v, %v; want %s", byEmail.ID, err, first.ID)
	}

	// Somebody else gets their own account.
	otherID := tgID + 1
	other, err := s.Adopt(ctx, account.Existing{TelegramID: &otherID, FirstName: "Bob", CreatedAt: now, UpdatedAt: now})
	if err != nil || other.ID == first.ID {
		t.Fatalf("another person = %v, %v", other.ID, err)
	}

	cleanup(t, p, other.ID)
}

func TestMergeFoldsTheLoserIntoTheSurvivor(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	// A Telegram-only account and, for the same person, an email-only one.
	survivor, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	email := "merge-" + uuid.NewString() + "@example.com"
	hash, _ := account.HashPassword("merge-password")

	loser, err := s.Adopt(ctx, account.Existing{Email: &email, PasswordHash: &hash, FirstName: "Web", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, survivor.ID, loser.ID)

	if err := s.Register(ctx, survivor.ID, account.Budget); err != nil {
		t.Fatal(err)
	}

	if err := s.Register(ctx, loser.ID, account.Learning); err != nil {
		t.Fatal(err)
	}

	merged, err := s.Merge(ctx, survivor.ID, loser.ID, now)
	if err != nil {
		t.Fatal(err)
	}

	if merged.ID != survivor.ID || merged.Email == nil || *merged.Email != email || merged.TelegramID == nil {
		t.Fatalf("merged = %+v", merged)
	}

	if _, err := s.SignInEmail(ctx, email, "merge-password"); err != nil {
		t.Fatalf("the loser's email login should now open the survivor: %v", err)
	}

	if _, err := s.ByID(ctx, loser.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("the loser account should be gone: %v", err)
	}

	// The survivor now holds data for both apps: purging only budget keeps it.
	if err := s.Register(ctx, survivor.ID, account.Budget); err != nil {
		t.Fatal(err)
	}

	if _, err := s.RequestDeletion(ctx, survivor.ID, now); err != nil {
		t.Fatal(err)
	}

	due := now.Add(time.Hour)

	if _, err := s.Purge(ctx, account.Budget, due, func(context.Context, []uuid.UUID) error { return nil }); err != nil {
		t.Fatal(err)
	}

	if _, err := s.ByID(ctx, survivor.ID); err != nil {
		t.Fatalf("learning still holds the survivor after the merge: %v", err)
	}
}

func TestAdoptGivesAnExistingAccountTheLoginsItLacks(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	tgID := tg(t).ID
	email := "late-" + uuid.NewString() + "@example.com"
	hash, _ := account.HashPassword("late-password")

	// The first app knew only the Telegram id …
	first, err := s.Adopt(ctx, account.Existing{TelegramID: &tgID, FirstName: "Ann", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, first.ID)

	// … the second one also had an email login for the same person.
	second, err := s.Adopt(ctx, account.Existing{
		TelegramID: &tgID, Email: &email, PasswordHash: &hash, FirstName: "Ann", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || second.ID != first.ID {
		t.Fatalf("second adopt = %v, %v; want %s", second.ID, err, first.ID)
	}

	if _, err := s.SignInEmail(ctx, email, "late-password"); err != nil {
		t.Fatalf("the email from the second app should open the shared account: %v", err)
	}

	// What the account has is not overwritten by a third source.
	other := "other-" + uuid.NewString() + "@example.com"
	otherHash, _ := account.HashPassword("other-password")

	third, err := s.Adopt(ctx, account.Existing{
		TelegramID: &tgID, Email: &other, PasswordHash: &otherHash, FirstName: "Ann", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || third.Email == nil || *third.Email != email {
		t.Fatalf("an existing email must stay: %+v, %v", third, err)
	}
}

func TestAdoptDoesNotTakeAnotherAccountsEmail(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()

	owner, err := s.SignInTelegram(ctx, tg(t), nil, now)
	if err != nil {
		t.Fatal(err)
	}

	email := "owned-" + uuid.NewString() + "@example.com"
	if _, err := s.LinkEmail(ctx, owner.ID, email, "owner-password", now); err != nil {
		t.Fatal(err)
	}

	otherTG := tg(t).ID + 7

	person, err := s.Adopt(ctx, account.Existing{TelegramID: &otherTG, FirstName: "Bob", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, owner.ID, person.ID)

	hash, _ := account.HashPassword("whatever-password")

	got, err := s.Adopt(ctx, account.Existing{TelegramID: &otherTG, Email: &email, PasswordHash: &hash, FirstName: "Bob", CreatedAt: now, UpdatedAt: now})
	if err != nil || got.ID != person.ID || got.Email != nil {
		t.Fatalf("must not take the other account's email: %+v, %v", got, err)
	}
}

func TestSignInTelegramFillsTheEmailLoginOfAnExistingAccount(t *testing.T) {
	s, p := store(t)
	ctx := context.Background()
	user := tg(t)

	acc, err := s.SignInTelegram(ctx, user, nil, now)
	if err != nil {
		t.Fatal(err)
	}

	cleanup(t, p, acc.ID)

	email := "carried-" + uuid.NewString() + "@example.com"
	hash, _ := account.HashPassword("carried-password")

	if _, err := s.SignInTelegram(ctx, user, &account.Credentials{Email: &email, PasswordHash: &hash}, now); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SignInEmail(ctx, email, "carried-password"); err != nil {
		t.Fatalf("a carried login should reach an account that already existed: %v", err)
	}
}
