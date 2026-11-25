package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/migrations"
)

// TestLearningLoopMigration runs 0036 up, down and up again on a scratch
// database that holds what the previous release had stored, and checks that
// none of it is touched and that the new tables appear and disappear whole.
func TestLearningLoopMigration(t *testing.T) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()

	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer admin.Close()

	name := "mig_loop_" + uuid.NewString()[:8]
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

	// The schema of the previous release, with a learner's program in it.
	if err := goose.UpToContext(ctx, sqlDB, ".", 35); err != nil {
		t.Fatalf("up to 35: %v", err)
	}

	legacy := seedLegacy(t, ctx, pool)

	if err := goose.UpToContext(ctx, sqlDB, ".", 36); err != nil {
		t.Fatalf("up to 36: %v", err)
	}

	legacy.check(t, ctx, pool, "after up")
	mustHaveLoopTables(t, ctx, pool, true)

	// The new tables take rows tied to the old teacher and refuse another learner's.
	insertLoopRows(t, ctx, pool, legacy)

	if err := goose.DownContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("down: %v", err)
	}

	if v, _ := goose.GetDBVersionContext(ctx, sqlDB); v != 35 {
		t.Fatalf("one down step left version %d, want 35", v)
	}

	legacy.check(t, ctx, pool, "after down")
	mustHaveLoopTables(t, ctx, pool, false)

	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("up again: %v", err)
	}

	legacy.check(t, ctx, pool, "after up again")
	mustHaveLoopTables(t, ctx, pool, true)
}

type legacyRows struct {
	user, teacher, path, topic, exercise, attempt uuid.UUID
}

func seedLegacy(t *testing.T, ctx context.Context, pool *pgxpool.Pool) legacyRows {
	t.Helper()

	q := db.New(pool)
	now := db.Time(time.Now().UTC())
	l := legacyRows{ids.New(), ids.New(), ids.New(), ids.New(), ids.New(), ids.New()}

	must := func(err error) {
		t.Helper()

		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	_, err := q.InsertUser(ctx, db.InsertUserParams{ID: ids.ToPg(l.user), FirstName: "Legacy", CreatedAt: now, UpdatedAt: now})
	must(err)
	_, err = q.InsertLearningTeacher(ctx, db.InsertLearningTeacherParams{ID: ids.ToPg(l.teacher), UserID: ids.ToPg(l.user), Language: "en", TranslationLanguage: "ru", CreatedAt: now})
	must(err)
	placement := ids.New()
	_, err = q.InsertLearningPathTest(ctx, db.InsertLearningPathTestParams{ID: ids.ToPg(placement), UserID: ids.ToPg(l.user), Kind: db.LearningPathTestKindPlacement, Language: "en", TranslationLanguage: "ru", Level: "A1", CreatedAt: now})
	must(err)
	_, err = q.InsertLearningPath(ctx, db.InsertLearningPathParams{ID: ids.ToPg(l.path), UserID: ids.ToPg(l.user), Language: "en", TranslationLanguage: "ru", Level: "A1", Summary: "s", Source: "test", PlacementTestID: ids.ToPg(placement), Preferences: []byte(`{}`), CreatedAt: now})
	must(err)
	_, err = q.InsertLearningPathTopic(ctx, db.InsertLearningPathTopicParams{ID: ids.ToPg(l.topic), PathID: ids.ToPg(l.path), Position: 0, Skill: "grammar", Title: "To be", Goal: "g", Vocabulary: []byte(`[]`), CreatedAt: now, SourceID: "a1-to-be"})
	must(err)
	_, err = q.InsertLearningPathExercise(ctx, db.InsertLearningPathExerciseParams{ID: ids.ToPg(l.exercise), TopicID: ids.ToPg(l.topic), Position: 0, Kind: db.LearningPathExerciseKindFill, Prompt: "I ... a student.", Options: []byte(`[]`), Answer: "am", CreatedAt: now})
	must(err)
	must(q.InsertLearningPathExerciseVersion(ctx, db.InsertLearningPathExerciseVersionParams{ExerciseID: ids.ToPg(l.exercise), TopicID: ids.ToPg(l.topic), Generation: 1}))
	_, err = q.InsertLearningPathAttempt(ctx, db.InsertLearningPathAttemptParams{ID: ids.ToPg(l.attempt), ExerciseID: ids.ToPg(l.exercise), UserID: ids.ToPg(l.user), Answer: "am", Correct: true, CreatedAt: now})
	must(err)

	return l
}

// check reads the legacy rows back untouched.
func (l legacyRows) check(t *testing.T, ctx context.Context, pool *pgxpool.Pool, when string) {
	t.Helper()

	for _, c := range []struct{ table, id string }{
		{"users", l.user.String()},
		{"learning_teachers", l.teacher.String()},
		{"learning_paths", l.path.String()},
		{"learning_path_topics", l.topic.String()},
		{"learning_path_exercises", l.exercise.String()},
		{"learning_path_attempts", l.attempt.String()},
	} {
		var n int
		if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = $1", c.table), c.id).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s: %s row %s: count %d, err %v", when, c.table, c.id, n, err)
		}
	}

	var source string
	if err := pool.QueryRow(ctx, "SELECT source_id FROM learning_path_topics WHERE id = $1", l.topic).Scan(&source); err != nil || source != "a1-to-be" {
		t.Fatalf("%s: the topic's source changed: %q, %v", when, source, err)
	}
}

func mustHaveLoopTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want bool) {
	t.Helper()

	for _, table := range []string{"kb_items", "learning_evidence", "learner_item_state", "learning_activities"} {
		var found *string
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::text", table).Scan(&found); err != nil {
			t.Fatal(err)
		}

		if (found != nil) != want {
			t.Fatalf("table %s present = %v, want %v", table, found != nil, want)
		}
	}
}

func insertLoopRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, l legacyRows) {
	t.Helper()

	item := ids.New()
	now := time.Now().UTC()

	if _, err := pool.Exec(ctx, `INSERT INTO kb_items (id, language, key, kind, skill, cefr_level, created_at) VALUES ($1, 'en', 'a1-to-be', 'grammar', 'grammar', 'A1', $2)`, item, now); err != nil {
		t.Fatalf("kb item: %v", err)
	}

	evidence := `INSERT INTO learning_evidence (id, user_id, teacher_id, item_id, skill, source, practice_mode, ref_type, ref_id, mode, outcome, score, weight, assisted, occurred_at)
		VALUES ($1, $2, $3, $4, 'grammar', 'learning_path', 'tracked', 'attempt', $5, 'recall', 'correct', 1, 1, true, $6)`

	if _, err := pool.Exec(ctx, evidence, ids.New(), l.user, l.teacher, item, l.attempt, now); err != nil {
		t.Fatalf("evidence: %v", err)
	}

	// The same attempt cannot give a second row.
	if _, err := pool.Exec(ctx, evidence, ids.New(), l.user, l.teacher, item, l.attempt, now); err == nil {
		t.Fatal("a second evidence row for one attempt was accepted")
	}

	// Another learner's id on this teacher is refused by the composite key.
	other := ids.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, first_name, created_at, updated_at) VALUES ($1, 'Other', $2, $2)`, other, now); err != nil {
		t.Fatalf("other user: %v", err)
	}

	if _, err := pool.Exec(ctx, evidence, ids.New(), other, l.teacher, item, ids.New(), now); err == nil {
		t.Fatal("evidence of one learner on another learner's teacher was accepted")
	}

	// Deleting the teacher takes its loop rows with it and leaves the program alone.
	if _, err := pool.Exec(ctx, "DELETE FROM learning_teachers WHERE id = $1", l.teacher); err != nil {
		t.Fatalf("delete teacher: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM learning_evidence WHERE teacher_id = $1", l.teacher).Scan(&n); err != nil || n != 0 {
		t.Fatalf("evidence left after its teacher: %d, %v", n, err)
	}

	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM learning_paths WHERE id = $1", l.path).Scan(&n); err != nil || n != 1 {
		t.Fatalf("the program went with the teacher: %d, %v", n, err)
	}

	// Put the teacher back so the legacy check of the caller still holds.
	if _, err := pool.Exec(ctx, "INSERT INTO learning_teachers (id, user_id, language, translation_language, created_at) VALUES ($1, $2, 'en', 'ru', $3)", l.teacher, l.user, now); err != nil {
		t.Fatalf("restore teacher: %v", err)
	}
}
