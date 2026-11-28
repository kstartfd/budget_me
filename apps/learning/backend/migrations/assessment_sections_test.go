package migrations_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/token"
	"learning_app/backend/migrations"
)

// This uses the old SQL contract throughout, including a prepared statement
// held open during migration and a transaction already writing when it starts.
func TestAssessmentSectionsMigrationAndOldWriter(t *testing.T) {
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
	name := "mig_sections_" + uuid.NewString()[:8]
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create isolated migration database: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)") })
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, ".", 38); err != nil {
		t.Fatal(err)
	}
	legacy := seedLegacy(t, ctx, pool)

	ids := make(map[string]uuid.UUID)
	users := make(map[string]uuid.UUID)
	for _, status := range []string{"in_progress", "paused", "completed", "partial"} {
		user, teacher, test := uuid.New(), uuid.New(), uuid.New()
		ids[status] = test
		users[status] = user
		mustExecSection(t, ctx, pool, `INSERT INTO users (id, first_name, created_at, updated_at) VALUES ($1, 'Example', now(), now())`, user)
		mustExecSection(t, ctx, pool, `INSERT INTO learning_teachers (id, user_id, language, translation_language, created_at) VALUES ($1,$2,'en','ru',now())`, teacher, user)
		terminal := status == "completed" || status == "partial"
		var result, reason any
		if status == "completed" {
			result = `{"sections":[{"skill":"grammar","status":"completed","usableEvidence":1}]}`
			reason = "enough_evidence"
		}
		if status == "partial" {
			result = `{"historical":"unchanged"}`
			reason = "learner_stopped"
		}
		mustExecSection(t, ctx, pool, `INSERT INTO learning_path_tests
		    (id,user_id,kind,language,translation_language,level,status,created_at,completed_at,
		     assessment_profile_id,assessment_type,assessment_status,start_band,assessment_policy,
		     assessment_explanation_language,assessment_batch,assessment_result,assessment_stop_reason)
		    VALUES ($1,$2,'placement','en','ru','A2', CASE WHEN $5 THEN 'done'::learning_path_test_status ELSE 'in_progress'::learning_path_test_status END,
		        now(),CASE WHEN $5 THEN now() END,$3,'quick_screening',$4,'A2',
		        '{"version":"grammar-quick-v1","batchSize":3,"minQuestions":6,"maxQuestions":12,"defaultBand":"A2","minCategories":3}',
		        'ru',1,$6,$7)`, test, user, teacher, status, terminal, result, reason)
		item, attempt := uuid.New(), uuid.New()
		mustExecSection(t, ctx, pool, `INSERT INTO learning_knowledge_items (id,language,key,kind,skill,cefr_level,created_at) VALUES ($1,'en',$2,'grammar','grammar','A2',now())`, item, "sample-"+status)
		mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_questions (test_id,question_key,level,position,skill,prompt,options,answer,assessment_item_id,assessment_batch) VALUES ($1,'q1','A2',0,'grammar','sample','[]','a',$2,1)`, test, item)
		mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_answers (test_id,question_key,answer,correct,assessment_attempt_id,assessment_outcome) VALUES ($1,'q1','a',true,$2,'correct')`, test, attempt)
		mustExecSection(t, ctx, pool, `INSERT INTO learning_evidence (id,user_id,teacher_id,item_id,skill,source,practice_mode,ref_type,ref_id,mode,outcome,score,weight,assisted,occurred_at) VALUES ($1,$2,$3,$4,'grammar','assessment','assessment','assessment_attempt',$1,'recognition','correct',1,0.4,false,now())`, attempt, user, teacher, item)
	}
	// A valid unanswered question is issued under the v38 schema, before any
	// section row exists. The new engine will resume this exact question.
	legacyItem := uuid.New()
	mustExecSection(t, ctx, pool, `INSERT INTO learning_knowledge_items (id,language,key,kind,skill,cefr_level,created_at) VALUES ($1,'en','a2-past-simple-irregular','grammar','grammar','A2',now())`, legacyItem)
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_questions
		(test_id,question_key,level,position,skill,prompt,options,answer,assessment_item_id,assessment_item_key,assessment_category,assessment_objective,assessment_format,assessment_version,assessment_origin,assessment_fingerprint,assessment_batch,assessment_alternatives)
		VALUES ($1,'legacy-q2','A2',1,'grammar','Yesterday I ___ it.','["found","find","finded","have found"]','found',$2,'a2-past-simple-irregular','verb_forms','Use past forms','multiple_choice','legacy-v1','curated','legacy-fingerprint',1,'{"answers":["found"]}')`, ids["in_progress"], legacyItem)

	old, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Release()
	if _, err := old.Conn().Prepare(ctx, "old_test", `SELECT * FROM learning_path_tests WHERE id=$1`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.Conn().Prepare(ctx, "old_questions", `SELECT * FROM learning_path_test_questions WHERE test_id=$1`); err != nil {
		t.Fatal(err)
	}
	oldTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer oldTx.Rollback(ctx)
	if _, err := oldTx.Exec(ctx, `UPDATE learning_path_tests SET level='B1', assessment_status='paused' WHERE id=$1`, ids["in_progress"]); err != nil {
		t.Fatal(err)
	}
	completed := make(chan error, 1)
	go func() { completed <- goose.UpToContext(ctx, db, ".", 39) }()
	select {
	case err := <-completed:
		t.Fatalf("migration did not wait for old transaction: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := oldTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-completed; err != nil {
		t.Fatalf("migration: %v", err)
	}

	for _, status := range []string{"in_progress", "paused", "completed", "partial"} {
		want := status
		if status == "in_progress" {
			want = "paused"
		}
		var id uuid.UUID
		var got string
		var answered int
		var noBlueprint, hasResult bool
		err := pool.QueryRow(ctx, `SELECT id,status,answered,blueprint_snapshot IS NULL,result_snapshot IS NOT NULL FROM learning_assessment_sections WHERE test_id=$1`, ids[status]).Scan(&id, &got, &answered, &noBlueprint, &hasResult)
		if err != nil || id != ids[status] || got != want || answered != 1 || !noBlueprint || hasResult != (status == "completed") {
			t.Fatalf("%s: id %s status %s answered %d legacy %v result %v: %v", status, id, got, answered, noBlueprint, hasResult, err)
		}
		var evidence int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning_assessment_section_evidence WHERE section_id=$1 AND skill='grammar'`, id).Scan(&evidence); err != nil || evidence != 1 {
			t.Fatalf("%s evidence: %d %v", status, evidence, err)
		}
	}
	legacy.check(t, ctx, pool, "section migration")
	rows, err := old.Query(ctx, "old_test", ids["paused"])
	if err != nil {
		t.Fatalf("pre-migration prepared test statement: %v", err)
	}
	rows.Close()
	// Resume and submit through the real new HTTP handlers after migration.
	// An unrelated account store is never queried by these session-only routes.
	const secret = "p316-legacy-test-secret"
	app := server.New(config.Config{AuthSecret: secret}, pool, pool)
	bearer := token.NewCodec(secret).Issue(users["in_progress"], time.Now(), time.Hour)
	path := "/api/v1/learning/path/tests/" + ids["in_progress"].String()
	call := func(method, suffix, body string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, path+suffix, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("legacy %s %s: %d %s", method, suffix, rec.Code, rec.Body.String())
		}
		var envelope struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	resumed := call("POST", "/assessment/resume", "{}")
	questions := resumed["questions"].([]any)
	if len(questions) != 1 || questions[0].(map[string]any)["key"] != "legacy-q2" {
		t.Fatalf("legacy question changed: %v", questions)
	}
	answered := call("POST", "/answers", `{"answers":[{"key":"legacy-q2","answer":"found"}]}`)
	if answered["assessment"].(map[string]any)["answered"] != float64(2) {
		t.Fatalf("legacy answers lost: %v", answered)
	}
	stopped := call("POST", "/assessment/complete", "{}")
	if stopped["assessment"].(map[string]any)["status"] != "partial" {
		t.Fatalf("legacy result: %v", stopped)
	}
	rows, err = old.Query(ctx, "old_questions", ids["paused"])
	if err != nil {
		t.Fatalf("pre-migration prepared questions statement: %v", err)
	}
	rows.Close()

	// Simulate the previous application issuing a new question and answer.
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_questions (test_id,question_key,level,position,skill,prompt,options,answer,assessment_batch) VALUES ($1,'q2','A2',1,'grammar','sample','[]','b',1)`, ids["paused"])
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_answers (test_id,question_key,answer,correct,assessment_attempt_id,assessment_outcome) VALUES ($1,'q2','b',true,$2,'correct')`, ids["paused"], uuid.New())
	mustExecSection(t, ctx, pool, `UPDATE learning_path_tests SET assessment_status='in_progress',level='B1',assessment_batch=2 WHERE id=$1`, ids["paused"])
	var state, band string
	var count, batch int
	if err := pool.QueryRow(ctx, `SELECT status,current_band,current_batch,answered FROM learning_assessment_sections WHERE test_id=$1`, ids["paused"]).Scan(&state, &band, &batch, &count); err != nil || state != "in_progress" || band != "B1" || batch != 2 || count != 2 {
		t.Fatalf("old writer not mirrored: %s %s %d %d: %v", state, band, batch, count, err)
	}
	// The old binary can also create an entire attempt after the migration.
	oldNew := uuid.New()
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_tests
		(id,user_id,kind,language,translation_language,level,status,created_at,assessment_profile_id,assessment_type,assessment_status,start_band,assessment_policy,assessment_explanation_language)
		SELECT $1,user_id,kind,language,translation_language,'A2','in_progress',now(),assessment_profile_id,'quick_screening','created','A2',assessment_policy,assessment_explanation_language
		FROM learning_path_tests WHERE id=$2`, oldNew, ids["completed"])
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_questions (test_id,question_key,level,position,skill,prompt,options,answer,assessment_batch) VALUES ($1,'q1','A2',0,'grammar','sample','[]','a',1)`, oldNew)
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_answers (test_id,question_key,answer,correct,assessment_attempt_id,assessment_outcome) VALUES ($1,'q1','a',true,$2,'correct')`, oldNew, uuid.New())
	if err := pool.QueryRow(ctx, `SELECT answered FROM learning_assessment_sections WHERE id=$1`, oldNew).Scan(&count); err != nil || count != 1 {
		t.Fatalf("old writer's new attempt was not mirrored: %d %v", count, err)
	}
	// The schema can retain independently progressing future sections, even
	// though the API does not yet enable Full Assessment or reading.
	future, grammar, reading, futureAttempt := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_tests
		(id,user_id,kind,language,translation_language,level,status,created_at,assessment_profile_id,assessment_type,assessment_status,start_band,assessment_policy,assessment_explanation_language)
		SELECT $1,user_id,kind,language,translation_language,'A2','in_progress',now(),assessment_profile_id,'full_assessment','created','A2',assessment_policy,assessment_explanation_language
		FROM learning_path_tests WHERE id=$2`, future, ids["partial"])
	mustExecSection(t, ctx, pool, `INSERT INTO learning_assessment_sections (id,test_id,skill,position,status,blueprint_version,policy_snapshot,blueprint_snapshot,current_band,answered,stop_reason,result_snapshot,created_at,completed_at)
		VALUES ($1,$3,'grammar',0,'completed','v1','{"max":4}','{"skill":"grammar"}','A2',1,'enough','{"score":1}',now(),now()),
		       ($2,$3,'reading',1,'paused','v2','{"max":6}','{"skill":"reading"}','B1',0,NULL,NULL,now(),NULL)`, grammar, reading, future)
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_questions (test_id,question_key,level,position,skill,prompt,options,answer) VALUES ($1,'grammar-q','A2',0,'grammar','sample','[]','a')`, future)
	if _, err := pool.Exec(ctx, `INSERT INTO learning_assessment_section_questions (test_id,question_key,skill,section_id) VALUES ($1,'grammar-q','reading',$2)`, future, reading); err == nil {
		t.Fatal("question linked to another skill")
	}
	mustExecSection(t, ctx, pool, `INSERT INTO learning_assessment_section_questions (test_id,question_key,skill,section_id) VALUES ($1,'grammar-q','grammar',$2)`, future, grammar)
	mustExecSection(t, ctx, pool, `INSERT INTO learning_path_test_answers (test_id,question_key,answer,correct,assessment_attempt_id,assessment_outcome) VALUES ($1,'grammar-q','a',true,$2,'correct')`, future, futureAttempt)
	mustExecSection(t, ctx, pool, `INSERT INTO learning_evidence (id,user_id,teacher_id,item_id,skill,source,practice_mode,ref_type,ref_id,mode,outcome,score,weight,assisted,occurred_at)
		SELECT $1,t.user_id,t.assessment_profile_id,q.assessment_item_id,'grammar','assessment','assessment','assessment_attempt',$1,'recognition','correct',1,0.4,false,now()
		FROM learning_path_tests t JOIN learning_path_test_questions q ON q.test_id=$2 AND q.question_key='q1' WHERE t.id=$3`, futureAttempt, ids["partial"], future)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning_assessment_section_evidence WHERE section_id=$1 AND skill='grammar' AND evidence_id=$2`, grammar, futureAttempt).Scan(&count); err != nil || count != 1 {
		t.Fatalf("future section evidence attribution: %d %v", count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM learning_assessment_sections WHERE id=$1`, reading).Scan(&state); err != nil || state != "paused" {
		t.Fatalf("other section state changed: %s %v", state, err)
	}
	if err := goose.DownContext(ctx, db, "."); err == nil {
		t.Fatal("rollback accepted independently stored sections")
	}
	mustExecSection(t, ctx, pool, `DELETE FROM learning_path_tests WHERE id=$1`, future)
	// The down guard prevents loss once a future section has independent data.
	mustExecSection(t, ctx, pool, `UPDATE learning_assessment_sections SET blueprint_snapshot='{"future":true}' WHERE test_id=$1`, ids["paused"])
	if err := goose.DownContext(ctx, db, "."); err == nil {
		t.Fatal("unsafe rollback accepted")
	}
	mustExecSection(t, ctx, pool, `UPDATE learning_assessment_sections SET blueprint_snapshot=NULL WHERE test_id=$1`, ids["paused"])
	// New Quick snapshots are also stored in the legacy policy JSON, so a safe
	// down/up round trip retains both the snapshot and stable section identifier.
	mustExecSection(t, ctx, pool, `UPDATE learning_path_tests SET assessment_policy=assessment_policy || '{"sectionBlueprint":{"version":"grammar-quick-v1","test":"retained"}}'::jsonb WHERE id=$1`, ids["paused"])
	if err := goose.DownContext(ctx, db, "."); err != nil {
		t.Fatalf("safe down: %v", err)
	}
	legacy.check(t, ctx, pool, "section rollback")
	var preserved string
	if err := pool.QueryRow(ctx, `SELECT assessment_result->>'historical' FROM learning_path_tests WHERE id=$1`, ids["partial"]).Scan(&preserved); err != nil || preserved != "unchanged" {
		t.Fatalf("historical result changed: %s %v", preserved, err)
	}
	if err := goose.UpToContext(ctx, db, ".", 39); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status,current_band,current_batch,answered FROM learning_assessment_sections WHERE id=$1`, ids["paused"]).Scan(&state, &band, &batch, &count); err != nil || state != "in_progress" || band != "B1" || batch != 2 || count != 2 {
		t.Fatalf("up again: %s %s %d %d: %v", state, band, batch, count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT blueprint_snapshot->>'test' FROM learning_assessment_sections WHERE id=$1`, ids["paused"]).Scan(&preserved); err != nil || preserved != "retained" {
		t.Fatalf("blueprint snapshot changed during rollback: %s %v", preserved, err)
	}
}

func mustExecSection(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
