package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var fs embed.FS

// Up brings the account database to the latest schema. Both apps call it on
// deploy; the advisory lock makes two deploys at once take turns instead of
// racing on CREATE TABLE.
func Up(ctx context.Context, dsn string) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open account database: %w", err)
	}

	defer func() { _ = conn.Close() }()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, conn, fs, goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)

	return err
}
