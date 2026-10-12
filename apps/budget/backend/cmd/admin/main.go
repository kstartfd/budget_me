// Command admin is a tiny operator CLI, run on the box the same way as migrate
// (docker compose --profile tools run --rm admin <cmd>). No HTTP, no roles —
// SSH access to the box IS the authorization. Raw pgx (like cmd/migrate uses
// raw goose): it is an entry point, not the app's commands/queries packages.
//
//	admin users                       — list accounts with their data counts
//	admin merge <survivorID> <loserID>— fold the loser account into the survivor
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		usage()
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail("DATABASE_URL is required")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fail("connect: " + err.Error())
	}

	defer func() { _ = conn.Close(ctx) }()

	switch os.Args[1] {
	case "users":
		listUsers(ctx, conn)
	case "merge":
		if len(os.Args) != 4 {
			fail("usage: admin merge <survivorID> <loserID>")
		}

		mergeUsers(ctx, conn, mustUUID(os.Args[2]), mustUUID(os.Args[3]))
	default:
		usage()
	}
}

// listUsers prints every account with its identities and how much data it owns —
// enough to pick the survivor and see whether the loser has anything to move.
func listUsers(ctx context.Context, conn *pgx.Conn) {
	rows, err := conn.Query(ctx, `
SELECT u.id, u.email, u.telegram_id, u.first_name, u.created_at,
       (SELECT count(*) FROM transactions t WHERE t.user_id = u.id),
       (SELECT count(*) FROM categories  c WHERE c.user_id = u.id)
FROM users u
ORDER BY u.created_at`)
	if err != nil {
		fail("query: " + err.Error())
	}

	defer rows.Close()

	fmt.Printf("%-36s  %-26s  %-13s  %-14s  %5s  %5s  %s\n",
		"id", "email", "telegram_id", "name", "txns", "cats", "created")

	for rows.Next() {
		var (
			id      string
			email   *string
			tg      *int64
			name    string
			created time.Time
			txns    int64
			cats    int64
		)

		if err := rows.Scan(&id, &email, &tg, &name, &created, &txns, &cats); err != nil {
			fail("scan: " + err.Error())
		}

		fmt.Printf("%-36s  %-26s  %-13s  %-14s  %5d  %5d  %s\n",
			id, str(email), tgStr(tg), name, txns, cats, created.Format("2006-01-02"))
	}

	if rows.Err() != nil {
		fail("rows: " + rows.Err().Error())
	}
}

// mergeUsers folds loser into survivor in one transaction: personal categories
// move over (names colliding with the survivor's are deduped — their
// transactions are remapped to the survivor's same-named category and the
// duplicate is dropped), all remaining transactions move over, the survivor
// inherits any identity (telegram / email) it was missing, and the loser row is
// deleted. Order matters: everything referencing the loser is moved BEFORE the
// delete (FKs are ON DELETE CASCADE), and the loser is deleted BEFORE the
// survivor claims its telegram_id/email (UNIQUE columns).
func mergeUsers(ctx context.Context, conn *pgx.Conn, survivor, loser string) {
	if survivor == loser {
		fail("survivor and loser must differ")
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		fail("begin: " + err.Error())
	}

	defer func() { _ = tx.Rollback(ctx) }()

	// Capture the loser's identity (also validates it exists).
	var (
		lTg    *int64
		lUser  *string
		lLang  *string
		lEmail *string
		lPw    *string
	)

	err = tx.QueryRow(ctx,
		`SELECT telegram_id, username, language_code, email, password_hash FROM users WHERE id = $1`,
		loser,
	).Scan(&lTg, &lUser, &lLang, &lEmail, &lPw)
	if err != nil {
		fail("loser not found: " + err.Error())
	}

	var survivorName string
	if err := tx.QueryRow(ctx, `SELECT first_name FROM users WHERE id = $1`, survivor).Scan(&survivorName); err != nil {
		fail("survivor not found: " + err.Error())
	}

	// 1. Remap transactions of loser categories whose name collides with a
	//    survivor category → point them at the survivor's category.
	remap, err := tx.Exec(ctx, `
UPDATE transactions t SET category_id = s.id
FROM categories lc
JOIN categories s ON s.user_id = $1 AND lower(s.name) = lower(lc.name)
WHERE lc.user_id = $2 AND t.category_id = lc.id`, survivor, loser)
	if err != nil {
		fail("remap transactions: " + err.Error())
	}

	// 2. Drop the now-orphaned duplicate loser categories.
	dropped, err := tx.Exec(ctx, `
DELETE FROM categories lc
WHERE lc.user_id = $2
  AND EXISTS (SELECT 1 FROM categories s WHERE s.user_id = $1 AND lower(s.name) = lower(lc.name))`,
		survivor, loser)
	if err != nil {
		fail("drop duplicate categories: " + err.Error())
	}

	// 3. Reassign the remaining (non-colliding) loser categories to the survivor.
	movedCats, err := tx.Exec(ctx, `UPDATE categories SET user_id = $1 WHERE user_id = $2`, survivor, loser)
	if err != nil {
		fail("move categories: " + err.Error())
	}

	// 4. Move all remaining loser transactions to the survivor.
	movedTx, err := tx.Exec(ctx, `UPDATE transactions SET user_id = $1 WHERE user_id = $2`, survivor, loser)
	if err != nil {
		fail("move transactions: " + err.Error())
	}

	// 5. Delete the loser (nothing references it now; frees its UNIQUE identity).
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, loser); err != nil {
		fail("delete loser: " + err.Error())
	}

	// 6. Survivor inherits any identity it was missing.
	if _, err := tx.Exec(ctx, `
UPDATE users SET
  telegram_id   = COALESCE(telegram_id, $2),
  username      = COALESCE(username, $3),
  language_code = COALESCE(language_code, $4),
  email         = COALESCE(email, $5),
  password_hash = COALESCE(password_hash, $6),
  updated_at    = $7
WHERE id = $1`, survivor, lTg, lUser, lLang, lEmail, lPw, time.Now()); err != nil {
		fail("attach identity: " + err.Error())
	}

	if err := tx.Commit(ctx); err != nil {
		fail("commit: " + err.Error())
	}

	fmt.Printf("merged %s → %s (%s): remapped %d tx, dropped %d dup categories, moved %d categories + %d tx, identity attached\n",
		loser, survivor, survivorName,
		remap.RowsAffected(), dropped.RowsAffected(), movedCats.RowsAffected(), movedTx.RowsAffected())
}

func mustUUID(s string) string {
	if _, err := uuid.Parse(s); err != nil {
		fail("not a valid uuid: " + s)
	}

	return s
}

func str(s *string) string {
	if s == nil {
		return "—"
	}

	return *s
}

func tgStr(v *int64) string {
	if v == nil {
		return "—"
	}

	return fmt.Sprintf("%d", *v)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:\n  admin users\n  admin merge <survivorID> <loserID>")
	os.Exit(2)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "admin:", msg)
	os.Exit(1)
}
