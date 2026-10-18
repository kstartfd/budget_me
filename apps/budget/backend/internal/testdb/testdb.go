// Package testdb is the shared Postgres harness for DB-backed tests.
//
// Behaviour that lives in SQL has to be tested against a real database: sqlc IS
// the repository here, so there is no interface to stand a mock in front of.
// Every such test needs the same two things — a connection that politely opts
// out when there is no database, and a throwaway user that takes its rows with
// it — so they live here once instead of in each test package.
package testdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// PresetExpenseCategory and PresetIncomeCategory are seeds from
// 0004_seed_categories.sql: shared presets (user_id NULL) visible to every user,
// with ids fixed across environments.
const (
	PresetExpenseCategory = "00000000-0000-4000-8000-000000000002" // groceries
	PresetIncomeCategory  = "00000000-0000-4000-8000-000000000007" // salary
)

// Clock pins "now" so a test can write into last month and read it back as if
// the calendar had already moved on.
type Clock struct{ At time.Time }

func (c Clock) Now() time.Time { return c.At }

// Pool connects using DATABASE_URL and SKIPS the test when it is unset. CI runs
// `go test ./...` with no database attached, and a skipped test there is the
// point — the suite stays green while these still run locally and against a
// real Postgres.
func Pool(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping DB-backed test")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool, db.New(pool)
}

// NewUser inserts a throwaway user and deletes it when the test ends; budgets,
// transactions and the rest follow via ON DELETE CASCADE, so tests never have to
// unwind what they wrote.
func NewUser(t *testing.T, pool *pgxpool.Pool, q *db.Queries, now time.Time, monthStartDay int) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	userID := ids.New()

	_, err := q.InsertUser(ctx, db.InsertUserParams{
		ID:            ids.ToPg(userID),
		FirstName:     "Test User",
		MonthStartDay: int16(monthStartDay),
		Currency:      db.CurrencyCodeBYN,
		Timezone:      "UTC",
		CreatedAt:     db.Time(now),
		UpdatedAt:     db.Time(now),
	})
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})

	return userID
}

// AccountPool connects to the shared account database (ACCOUNT_DATABASE_URL,
// migrated with budget_me/account/migrations) and SKIPS the test when it is unset.
func AccountPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("ACCOUNT_DATABASE_URL")
	if url == "" {
		t.Skip("ACCOUNT_DATABASE_URL not set — skipping account-backed test")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect account database: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
