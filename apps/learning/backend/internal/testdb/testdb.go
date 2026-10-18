package testdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type Clock struct{ At time.Time }

func (c Clock) Now() time.Time { return c.At }

func Pool(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, db.New(pool)
}

func NewUser(t *testing.T, pool *pgxpool.Pool, q *db.Queries, now time.Time, _ int) uuid.UUID {
	t.Helper()
	userID := ids.New()
	_, err := q.InsertUser(context.Background(), db.InsertUserParams{
		ID: ids.ToPg(userID), FirstName: "Test User", CreatedAt: db.Time(now), UpdatedAt: db.Time(now),
	})
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID) })
	return userID
}

// AccountPool connects to the shared account database (ACCOUNT_DATABASE_URL,
// migrated with budget_me/account/migrations) and SKIPS the test when it is unset.
func AccountPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("ACCOUNT_DATABASE_URL")
	if url == "" {
		t.Skip("ACCOUNT_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect account database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
