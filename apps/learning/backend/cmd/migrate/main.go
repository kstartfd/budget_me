package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"learning_app/backend/migrations"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|status|reset|version>")
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
	case "down":
		err = goose.DownContext(ctx, d, ".")
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
