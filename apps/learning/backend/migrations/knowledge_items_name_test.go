package migrations_test

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"learning_app/backend/internal/ids"
	"learning_app/backend/migrations"
)

// TestKnowledgeItemTableRename proves that the previous application can still
// read and upsert through kb_items while the new application uses the renamed
// table, with the same IDs and existing foreign keys across up/down/up.
func TestKnowledgeItemTableRename(t *testing.T) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := "mig_knowledge_name_" + uuid.NewString()[:8]
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Skipf("cannot create a scratch database (needs CREATEDB): %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)") })
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	sqlDB, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, sqlDB, ".", 37); err != nil {
		t.Fatalf("up to 37: %v", err)
	}
	legacy := seedLegacy(t, ctx, pool)
	item := ids.New()
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx, `INSERT INTO kb_items (id, language, key, kind, skill, cefr_level, created_at)
		VALUES ($1, 'en', 'a1-to-be', 'grammar', 'grammar', 'A1', $2)`, item, now); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO learning_evidence
		(id, user_id, teacher_id, item_id, skill, source, practice_mode, ref_type, ref_id,
		mode, outcome, score, weight, assisted, occurred_at)
		VALUES ($1, $2, $3, $4, 'grammar', 'learning_path', 'tracked', 'attempt', $5,
		'recall', 'correct', 1, 1, false, $6)`, ids.New(), legacy.user, legacy.teacher, item, legacy.attempt, now); err != nil {
		t.Fatalf("seed foreign key: %v", err)
	}
	// The previous release may hold a server-side prepared statement on an open
	// connection while the deploy applies the migration.
	oldUpsert := `INSERT INTO kb_items (id, language, key, kind, skill, cefr_level, unit, created_at)
		VALUES ($1, 'en', $2, 'grammar', 'grammar', 'A1', '', $3)
		ON CONFLICT (language, key) DO UPDATE SET key = kb_items.key RETURNING *`
	oldConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer oldConn.Release()
	if _, err := oldConn.Conn().Prepare(ctx, "legacy_kb_upsert", oldUpsert); err != nil {
		t.Fatalf("prepare old upsert before migration: %v", err)
	}
	if err := goose.UpToContext(ctx, sqlDB, ".", 38); err != nil {
		t.Fatalf("up to 38: %v", err)
	}
	checkKnowledgeNames(t, ctx, pool, item, "r", "v")
	legacy.check(t, ctx, pool, "after rename")

	// This is the exact old application's conflict clause, including its table
	// qualifier. A simple view must preserve it during a blue-green release.
	var preparedID uuid.UUID
	var preparedLanguage, preparedKey, preparedKind, preparedSkill, preparedLevel, preparedUnit string
	var preparedCreatedAt time.Time
	if err := oldConn.QueryRow(ctx, "legacy_kb_upsert", ids.New(), "a1-to-be", now).Scan(
		&preparedID, &preparedLanguage, &preparedKey, &preparedKind, &preparedSkill,
		&preparedLevel, &preparedUnit, &preparedCreatedAt,
	); err != nil || preparedID != item {
		t.Fatalf("prepared old upsert after migration: id %s, want %s: %v", preparedID, item, err)
	}
	newUpsert := `INSERT INTO learning_knowledge_items (id, language, key, kind, skill, cefr_level, unit, created_at)
		VALUES ($1, 'en', $2, 'grammar', 'grammar', 'A1', '', $3)
		ON CONFLICT (language, key) DO UPDATE SET key = learning_knowledge_items.key RETURNING *`
	for _, tc := range []struct{ name, query, key string }{
		{"old existing", oldUpsert, "a1-to-be"},
		{"old new", oldUpsert, "a1-articles"},
		{"new existing", newUpsert, "a1-to-be"},
		{"new after old", newUpsert, "a1-articles"},
		{"new new", newUpsert, "a1-plurals"},
		{"old after new", oldUpsert, "a1-plurals"},
	} {
		var got uuid.UUID
		var language, key, kind, skill, cefrLevel, unit string
		var createdAt time.Time
		if err := pool.QueryRow(ctx, tc.query, ids.New(), tc.key, now).Scan(
			&got, &language, &key, &kind, &skill, &cefrLevel, &unit, &createdAt,
		); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if language != "en" || key != tc.key || kind != "grammar" || skill != "grammar" || cefrLevel != "A1" || unit != "" {
			t.Fatalf("%s returned wrong item fields", tc.name)
		}
		if tc.key == "a1-to-be" && got != item {
			t.Fatalf("%s returned %s, want preserved %s", tc.name, got, item)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning_knowledge_items`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("renamed table has %d items, want 3: %v", count, err)
	}
	if err := goose.DownContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("down to 37: %v", err)
	}
	checkKnowledgeNames(t, ctx, pool, item, "", "r")
	legacy.check(t, ctx, pool, "after rollback")
	if err := goose.UpToContext(ctx, sqlDB, ".", 38); err != nil {
		t.Fatalf("up again: %v", err)
	}
	checkKnowledgeNames(t, ctx, pool, item, "r", "v")
}

func checkKnowledgeNames(t *testing.T, ctx context.Context, pool *pgxpool.Pool, item uuid.UUID, newKind, oldKind string) {
	t.Helper()
	for _, tc := range []struct{ name, kind string }{
		{"learning_knowledge_items", newKind}, {"kb_items", oldKind},
	} {
		var got *string
		if err := pool.QueryRow(ctx, `SELECT (SELECT relkind::text FROM pg_class WHERE oid = to_regclass($1))`, tc.name).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if tc.kind == "" && got != nil || tc.kind != "" && (got == nil || *got != tc.kind) {
			t.Fatalf("%s kind = %v, want %q", tc.name, got, tc.kind)
		}
	}
	var ref string
	if err := pool.QueryRow(ctx, `SELECT confrelid::regclass::text FROM pg_constraint
		WHERE conname = 'learning_evidence_item_id_fkey'`).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	want := "learning_knowledge_items"
	if newKind == "" {
		want = "kb_items"
	}
	if ref != want {
		t.Fatalf("evidence FK points to %s, want %s", ref, want)
	}
	var evidenceItem uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT item_id FROM learning_evidence WHERE item_id = $1`, item).Scan(&evidenceItem); err != nil || evidenceItem != item {
		t.Fatalf("evidence item = %s, want %s: %v", evidenceItem, item, err)
	}
}
