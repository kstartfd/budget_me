package account_test

import (
	"context"
	"os"
	"testing"

	"budget_me/account/migrations"
)

// TestMain brings the schema up through the same entry point the apps use, so
// the migration itself is exercised by every run that has a database.
func TestMain(m *testing.M) {
	if url := os.Getenv("ACCOUNT_DATABASE_URL"); url != "" {
		if err := migrations.Up(context.Background(), url); err != nil {
			panic(err)
		}
	}

	os.Exit(m.Run())
}
