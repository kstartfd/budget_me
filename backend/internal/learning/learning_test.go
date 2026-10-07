package learning_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/config"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/learning/commands"
	"budget_me/backend/internal/server"
	"budget_me/backend/internal/testdb"
	"budget_me/backend/internal/token"
)

const authSecret = "learning-integration-test-secret-long-enough"

func request(t *testing.T, e *echo.Echo, method, path, raw string, body any, key string) (int, map[string]any) {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+raw)

	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("%s %s: %s", method, path, rec.Body.String())
	}

	return rec.Code, response
}
func data(t *testing.T, response map[string]any) map[string]any {
	t.Helper()

	d, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected response: %v", response)
	}

	return d
}

func setup(t *testing.T) (*echo.Echo, *db.Queries, uuid.UUID, string, commands.ProcessHandler) {
	t.Helper()
	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)
	raw := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)
	e := server.New(config.Config{AuthSecret: authSecret, UploadsDir: t.TempDir()}, pool)

	return e, q, user, raw, commands.NewProcessHandler(pool, clock.New())
}
func topic(t *testing.T, e *echo.Echo, raw string) string {
	t.Helper()

	code, body := request(t, e, "POST", "/api/v1/learning/topics", raw, map[string]any{"name": "English", "language": "en", "translationLanguage": "ru"}, "")
	if code != http.StatusCreated {
		t.Fatalf("topic: %d %v", code, body)
	}

	return data(t, body)["id"].(string)
}
func apiToken(t *testing.T, e *echo.Echo, raw, scope, topicID string) (string, string) {
	t.Helper()

	code, body := request(t, e, "POST", "/api/v1/learning/tokens", raw, map[string]any{"name": "Chrome", "scope": scope, "topicId": topicID}, "")
	if code != 201 {
		t.Fatalf("token: %d %v", code, body)
	}

	d := data(t, body)

	return d["id"].(string), d["token"].(string)
}

func TestCaptureLifecycleScopesAndOwnership(t *testing.T) {
	e, q, user, session, processor := setup(t)
	topicID := topic(t, e, session)
	tokenID, raw := apiToken(t, e, session, "capture", topicID)
	fields := map[string]any{"topicId": topicID, "kind": "word", "text": "  serendipity  ", "source": "  Футурама · сезон 2 · Фрай  ", "explanation": "  Неожиданная удачная находка.  "}

	code, body := request(t, e, "POST", "/api/v1/learning/captures", raw, fields, "capture-1")
	if code != 202 || data(t, body)["status"] != "pending" {
		t.Fatalf("enqueue: %d %v", code, body)
	}

	jobID := data(t, body)["id"].(string)

	code, again := request(t, e, "POST", "/api/v1/learning/captures", raw, fields, "capture-1")
	if code != 202 || data(t, again)["id"] != jobID {
		t.Fatalf("not idempotent: %d %v", code, again)
	}

	fields["source"] = "Другой сериал"
	code, _ = request(t, e, "POST", "/api/v1/learning/captures", raw, fields, "capture-1")
	if code != 409 {
		t.Fatalf("source must participate in idempotency: %d", code)
	}
	fields["source"] = "  Футурама · сезон 2 · Фрай  "

	fields["text"] = "another word"

	code, _ = request(t, e, "POST", "/api/v1/learning/captures", raw, fields, "capture-1")
	if code != 409 {
		t.Fatalf("idempotency conflict: %d", code)
	}

	for _, path := range []string{"/api/v1/transactions", "/api/v1/learning/tokens"} {
		code, _ = request(t, e, "GET", path, raw, nil, "")
		if code != 401 {
			t.Fatalf("token escaped Learning at %s: %d", path, code)
		}
	}

	code, _ = request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID, raw, nil, "")
	if code != 403 {
		t.Fatalf("capture token read entries: %d", code)
	}

	code, _ = request(t, e, "GET", "/api/v1/learning/topics", raw, nil, "")
	if code != 200 {
		t.Fatalf("capture token cannot discover topics: %d", code)
	}

	worked, err := processor.Handle(context.Background())
	if err != nil || !worked {
		t.Fatalf("process: %t %v", worked, err)
	}

	code, body = request(t, e, "GET", "/api/v1/learning/captures/"+jobID, raw, nil, "")
	if code != 200 || data(t, body)["status"] != "done" {
		t.Fatalf("capture status: %d %v", code, body)
	}

	entryID := data(t, body)["entryId"].(string)
	code, body = request(t, e, "GET", "/api/v1/learning/entries/"+entryID, session, nil, "")

	entry := data(t, body)
	if code != 200 || entry["text"] != "serendipity" || entry["status"] != "inbox" || entry["translation"] != "" || entry["source"] != "Футурама · сезон 2 · Фрай" || entry["explanation"] != "Неожиданная удачная находка." {
		t.Fatalf("entry: %d %v", code, entry)
	}

	code, body = request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID+"&search=Фрай", session, nil, "")
	if code != 200 || len(body["data"].([]any)) != 1 {
		t.Fatalf("search by source: %d %v", code, body)
	}

	_, rw := apiToken(t, e, session, "read_write", topicID)
	entry["translation"] = "счастливая случайность"
	entry["nativeUsage"] = "a happy accident"
	entry["example"] = "Finding this place was pure serendipity."
	entry["rule"] = "noun"
	entry["status"] = "learning"

	code, body = request(t, e, "PUT", "/api/v1/learning/entries/"+entryID, rw, entry, "")
	if code != 200 || data(t, body)["status"] != "learning" || data(t, body)["source"] != "Футурама · сезон 2 · Фрай" || data(t, body)["explanation"] != "Неожиданная удачная находка." {
		t.Fatalf("edit: %d %v", code, body)
	}

	code, _ = request(t, e, "POST", "/api/v1/learning/entries", rw, fields, "")
	if code != 403 {
		t.Fatalf("token bypassed capture queue: %d", code)
	}

	otherSession := token.NewCodec(authSecret).Issue(ids.New(), time.Now().UTC(), time.Hour)
	for _, path := range []string{"/api/v1/learning/captures/" + jobID, "/api/v1/learning/entries/" + entryID, "/api/v1/learning/entries?topicId=" + topicID} {
		code, _ = request(t, e, "GET", path, otherSession, nil, "")
		if code != 404 {
			t.Fatalf("other user read %s: %d", path, code)
		}
	}

	code, _ = request(t, e, "POST", "/api/v1/learning/captures", otherSession, fields, "")
	if code != 404 {
		t.Fatalf("other user enqueued into topic: %d", code)
	}

	code, _ = request(t, e, "DELETE", "/api/v1/learning/tokens/"+tokenID, otherSession, nil, "")
	if code != 404 {
		t.Fatalf("other user revoked token: %d", code)
	}

	code, body = request(t, e, "GET", "/api/v1/learning/tokens", session, nil, "")
	if code != 200 {
		t.Fatal(body)
	}

	for _, item := range body["data"].([]any) {
		if _, ok := item.(map[string]any)["token"]; ok {
			t.Fatal("secret leaked in token list")
		}
	}

	code, _ = request(t, e, "DELETE", "/api/v1/learning/tokens/"+tokenID, session, nil, "")
	if code != 200 {
		t.Fatalf("revoke: %d", code)
	}

	code, _ = request(t, e, "GET", "/api/v1/learning/captures/"+jobID, raw, nil, "")
	if code != 401 {
		t.Fatalf("revoked token accepted: %d", code)
	}

	code, _ = request(t, e, "DELETE", "/api/v1/learning/entries/"+entryID, rw, nil, "")
	if code != 200 {
		t.Fatalf("delete: %d", code)
	}

	topics, err := q.ListLearningTopics(context.Background(), ids.ToPg(user))
	if err != nil || len(topics) != 1 {
		t.Fatalf("topics: %v %v", topics, err)
	}
}

func TestCaptureBurstAndConcurrentWorkers(t *testing.T) {
	e, q, user, session, processor := setup(t)
	topicID := topic(t, e, session)
	_, raw := apiToken(t, e, session, "capture", topicID)

	var wg sync.WaitGroup
	for i := range 40 {
		wg.Go(func() {
			key := fmt.Sprintf("burst-%d", i)

			fields := map[string]any{"topicId": topicID, "kind": "phrase", "text": fmt.Sprintf("phrase %d", i)}
			for range 2 {
				code, body := request(t, e, "POST", "/api/v1/learning/captures", raw, fields, key)
				if code != 202 {
					t.Errorf("enqueue: %d %v", code, body)
				}
			}
		})
	}

	wg.Wait()

	for range 4 {
		wg.Go(func() {
			for {
				worked, err := processor.Handle(context.Background())
				if err != nil {
					t.Errorf("worker: %v", err)
					return
				}

				if !worked {
					return
				}
			}
		})
	}

	wg.Wait()

	rows, err := q.ListLearningEntries(context.Background(), db.ListLearningEntriesParams{UserID: ids.ToPg(user), TopicID: ids.ToPg(uuid.MustParse(topicID)), PageLimit: 100})
	if err != nil || len(rows) != 40 {
		t.Fatalf("burst created %d entries, want 40: %v", len(rows), err)
	}

	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.Text] {
			t.Fatalf("duplicate %q", row.Text)
		}

		seen[row.Text] = true
	}
}

func TestCaptureRecoversExpiredLeaseAndFailsDeletedTopic(t *testing.T) {
	pool, q := testdb.Pool(t)
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	user := testdb.NewUser(t, pool, q, now, 1)
	clk := testdb.Clock{At: now}
	topics := commands.NewTopicHandler(q, clk)

	row, err := topics.Handle(context.Background(), commands.SaveTopic{UserID: user, Name: "English", Language: "en", TranslationLanguage: "ru"})
	if err != nil {
		t.Fatal(err)
	}

	enqueue := commands.NewCaptureHandler(q, clk)

	job, err := enqueue.Handle(context.Background(), commands.EnqueueCapture{UserID: user, RequestKey: "crashed", Fields: commands.EntryFields{TopicID: ids.FromPg(row.ID), Text: "hit the road"}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = q.ClaimLearningCapture(context.Background(), db.ClaimLearningCaptureParams{Now: db.Time(now), LockedUntil: db.Time(now.Add(time.Minute)), LeaseID: ids.ToPg(ids.New())})
	if err != nil {
		t.Fatal(err)
	}

	processor := commands.NewProcessHandler(pool, testdb.Clock{At: now.Add(2 * time.Minute)})

	worked, err := processor.Handle(context.Background())
	if err != nil || !worked {
		t.Fatalf("recover lease: %v", err)
	}

	done, err := q.GetLearningCapture(context.Background(), db.GetLearningCaptureParams{ID: job.ID, UserID: ids.ToPg(user)})
	if err != nil || done.Status != db.LearningCaptureStatusDone || done.Attempts != 2 {
		t.Fatalf("recovery: %v %v", done, err)
	}

	doomed, err := enqueue.Handle(context.Background(), commands.EnqueueCapture{UserID: user, RequestKey: "deleted-topic", Fields: commands.EntryFields{TopicID: ids.FromPg(row.ID), Text: "goodbye"}})
	if err != nil {
		t.Fatal(err)
	}

	if err := topics.Delete(context.Background(), user, ids.FromPg(row.ID)); err != nil {
		t.Fatal(err)
	}

	_, err = processor.Handle(context.Background())
	if err == nil {
		t.Fatal("deleted topic processed successfully")
	}

	failed, err := q.GetLearningCapture(context.Background(), db.GetLearningCaptureParams{ID: doomed.ID, UserID: ids.ToPg(user)})
	if err != nil || failed.Status != db.LearningCaptureStatusFailed {
		t.Fatalf("deleted-topic status: %v %v", failed, err)
	}
}

func TestInvalidCaptureDoesNotEnterQueue(t *testing.T) {
	e, _, _, session, _ := setup(t)

	topicID := topic(t, e, session)
	for _, fields := range []map[string]any{
		{"topicId": topicID, "text": "   "},
		{"topicId": topicID, "text": "hello", "kind": "rule"},
		{"topicId": topicID, "text": "hello", "status": "wrong"},
		{"topicId": "bad", "text": "hello"},
	} {
		code, _ := request(t, e, "POST", "/api/v1/learning/captures", session, fields, "")
		if code != 400 {
			t.Fatalf("invalid payload accepted: %v status %d", fields, code)
		}
	}
}

func TestCaptureReplayAfterTopicDeletion(t *testing.T) {
	e, _, _, session, processor := setup(t)
	topicID := topic(t, e, session)
	_, raw := apiToken(t, e, session, "capture", topicID)
	fields := map[string]any{"topicId": topicID, "text": "  hello  "}

	code, body := request(t, e, "POST", "/api/v1/learning/captures", raw, fields, "replay-deleted")
	if code != 202 {
		t.Fatalf("enqueue: %d %v", code, body)
	}

	jobID := data(t, body)["id"]
	if _, err := processor.Handle(context.Background()); err != nil {
		t.Fatal(err)
	}

	code, _ = request(t, e, "DELETE", "/api/v1/learning/topics/"+topicID, session, nil, "")
	if code != 200 {
		t.Fatalf("delete topic: %d", code)
	}

	// Deleting the topic revokes its scoped token; the account session can still
	// replay the original receipt without creating a new entry.
	replacement := session
	fields["text"] = "hello"

	code, body = request(t, e, "POST", "/api/v1/learning/captures", replacement, fields, "replay-deleted")
	if code != 202 || data(t, body)["id"] != jobID || data(t, body)["status"] != "done" {
		t.Fatalf("replay after deletion: %d %v", code, body)
	}

	fields["text"] = "different"

	code, _ = request(t, e, "POST", "/api/v1/learning/captures", replacement, fields, "replay-deleted")
	if code != 409 {
		t.Fatalf("changed replay: %d", code)
	}

	code, _ = request(t, e, "POST", "/api/v1/learning/captures", replacement, fields, "new-key")
	if code != 404 {
		t.Fatalf("new capture into deleted topic: %d", code)
	}
}

func TestUpdateWithNilIDDoesNotCreateResource(t *testing.T) {
	e, q, user, session, _ := setup(t)

	topicID := topic(t, e, session)
	for _, resource := range []struct {
		path string
		body map[string]any
	}{
		{"topics", map[string]any{"name": "Unexpected", "language": "en", "translationLanguage": "ru"}},
		{"entries", map[string]any{"topicId": topicID, "text": "Unexpected"}},
	} {
		code, _ := request(t, e, "PUT", "/api/v1/learning/"+resource.path+"/"+uuid.Nil.String(), session, resource.body, "")
		if code != 400 {
			t.Fatalf("PUT %s with nil ID: %d", resource.path, code)
		}
	}

	rows, err := q.ListLearningTopics(context.Background(), ids.ToPg(user))
	if err != nil || len(rows) != 1 {
		t.Fatalf("unexpected topics: %v %v", rows, err)
	}

	code, body := request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID, session, nil, "")
	if code != 200 || len(body["data"].([]any)) != 0 {
		t.Fatalf("unexpected entries: %d %v", code, body)
	}
}

func TestEntryPaginationAndFilters(t *testing.T) {
	e, q, user, session, _ := setup(t)
	topicID := topic(t, e, session)
	now := time.Now().UTC()

	for i := range 105 {
		text, kind, status := "ordinary phrase", db.LearningEntryKindPhrase, db.LearningEntryStatusKnown
		if i == 0 {
			text, kind, status = "needle%", db.LearningEntryKindWord, db.LearningEntryStatusLearning
		}

		_, err := q.InsertLearningEntry(context.Background(), db.InsertLearningEntryParams{ID: ids.ToPg(ids.New()), UserID: ids.ToPg(user), TopicID: ids.ToPg(uuid.MustParse(topicID)), Text: text, Kind: kind, Status: status, CreatedAt: db.Time(now.Add(time.Duration(i) * time.Second)), UpdatedAt: db.Time(now)})
		if err != nil {
			t.Fatal(err)
		}
	}

	code, body := request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID, session, nil, "")
	if code != 200 || len(body["data"].([]any)) != 100 {
		t.Fatalf("first page: %d %v", code, body)
	}

	code, body = request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID+"&offset=100", session, nil, "")
	if code != 200 || len(body["data"].([]any)) != 5 {
		t.Fatalf("second page: %d %v", code, body)
	}

	code, body = request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID+"&kind=word&status=learning&search=NEEDLE%25", session, nil, "")
	if code != 200 || len(body["data"].([]any)) != 1 {
		t.Fatalf("filtered search beyond first page: %d %v", code, body)
	}

	for _, suffix := range []string{"&kind=bad", "&status=bad", "&limit=201", "&offset=-1"} {
		code, _ = request(t, e, "GET", "/api/v1/learning/entries?topicId="+topicID+suffix, session, nil, "")
		if code != 400 {
			t.Fatalf("invalid filter %s: %d", suffix, code)
		}
	}
}
