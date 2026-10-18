package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	ident "budget_me/account"
	accountmigrations "budget_me/account/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/migrations"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|down-to N|status|reset|version>")
		os.Exit(2)
	}

	cmd := os.Args[1]

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	d, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}

	defer func() { _ = d.Close() }()

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Fprintln(os.Stderr, "dialect:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch cmd {
	case "up":
		err = goose.UpContext(ctx, d, ".")
		// The shared account schema is migrated by whichever app deploys first.
		if url := os.Getenv("ACCOUNT_DATABASE_URL"); err == nil && url != "" {
			err = accountmigrations.Up(ctx, url)
		}
	case "backfill-accounts":
		// Links every user from before accounts existed to its account (creating
		// it, or joining the one the other app already made). Idempotent.
		var n int

		n, err = backfillAccounts(ctx)
		if err == nil {
			fmt.Println("backfill-accounts: linked", n)
		}
	case "down":
		err = goose.DownContext(ctx, d, ".")
	case "down-to":
		// Blue-green rollback (infra/budget/scripts/bluegreen.sh): bring the
		// schema back to the version a previous release was deployed with. Run
		// from the NEWER image — only it carries the Down SQL of its migrations.
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: migrate down-to <version>")
			os.Exit(2)
		}

		var v int64

		v, err = strconv.ParseInt(os.Args[2], 10, 64)
		if err == nil {
			err = goose.DownToContext(ctx, d, ".", v)
		}
	case "status":
		err = goose.StatusContext(ctx, d, ".")
	case "reset":
		err = goose.ResetContext(ctx, d, ".")
	case "version":
		var v int64

		v, err = goose.GetDBVersionContext(ctx, d)
		if err == nil {
			fmt.Println("version:", v)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func backfillAccounts(ctx context.Context) (int, error) {
	url := os.Getenv("ACCOUNT_DATABASE_URL")
	if url == "" {
		return 0, fmt.Errorf("ACCOUNT_DATABASE_URL is required")
	}

	appPool, err := db.NewPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return 0, err
	}

	defer appPool.Close()

	accountPool, err := db.NewPool(ctx, url)
	if err != nil {
		return 0, err
	}

	defer accountPool.Close()

	return accountlink.New(db.New(appPool), ident.New(accountPool), clock.New()).Backfill(ctx)
}
