package learning_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningaudio"
	"learning_app/backend/internal/token"
)

type ttsStub struct {
	mu      sync.Mutex
	status  int
	body    []byte
	queries []string
}

func (s *ttsStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queries = append(s.queries, r.URL.RawQuery)

	if s.status != http.StatusOK {
		http.Error(w, "tts failed", s.status)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	_, _ = w.Write(s.body)
}

func (s *ttsStub) setStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func (s *ttsStub) requestQueries() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.queries...)
}

func setupAudioEnv(t *testing.T) (aiEnv, *ttsStub, string) {
	t.Helper()

	tts := &ttsStub{status: http.StatusOK, body: []byte("cached mp3 payload")}
	srv := httptest.NewServer(tts)
	t.Cleanup(srv.Close)

	audioDir := t.TempDir()
	env := setupAIWith(t, func(cfg *config.Config) {
		cfg.TTSURL = srv.URL
		cfg.AudioCacheDir = audioDir
	})

	return env, tts, audioDir
}

func createEntry(t *testing.T, e *echo.Echo, session, topicID, text string) string {
	t.Helper()

	code, body := request(t, e, "POST", "/api/v1/learning/entries", session, map[string]any{"topicId": topicID, "kind": "word", "text": text}, "")
	if code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	return data(t, body)["id"].(string)
}

func getEntryAudio(t *testing.T, e *echo.Echo, session, entryID, ifNoneMatch string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/learning/entries/"+entryID+"/audio", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+session)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return rec
}

func TestEntryAudioSynthesizesOnceAndCachesByLanguageAndText(t *testing.T) {
	env, tts, audioDir := setupAudioEnv(t)
	topicID := topic(t, env.e, env.session)
	text := "audio cache " + uuid.NewString()
	entryID := createEntry(t, env.e, env.session, topicID, text)

	rec := getEntryAudio(t, env.e, env.session, entryID, "")
	if rec.Code != http.StatusOK || rec.Body.String() != string(tts.body) {
		t.Fatalf("audio: %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Fatalf("cache control = %q", got)
	}

	queries := tts.requestQueries()
	if len(queries) != 1 {
		t.Fatalf("provider calls = %d: %v", len(queries), queries)
	}

	values, err := url.ParseQuery(queries[0])
	if err != nil || values.Get("q") != text || values.Get("tl") != "en" || values.Get("client") != "tw-ob" {
		t.Fatalf("provider query = %v (%v)", values, err)
	}

	row, err := env.q.GetLearningAudioCache(context.Background(), db.GetLearningAudioCacheParams{Language: "en", TextHash: learningaudio.TextHash(text)})
	if err != nil {
		t.Fatal(err)
	}
	if row.Text != text || row.ContentType != "audio/mpeg" || row.ByteSize != int64(len(tts.body)) {
		t.Fatalf("cache row: %+v", row)
	}
	if _, err := os.Stat(filepath.Join(audioDir, row.StoragePath)); err != nil {
		t.Fatalf("audio file: %v", err)
	}

	second := getEntryAudio(t, env.e, env.session, entryID, "")
	if second.Code != http.StatusOK || second.Body.String() != string(tts.body) {
		t.Fatalf("cached audio: %d %q", second.Code, second.Body.String())
	}
	if got := len(tts.requestQueries()); got != 1 {
		t.Fatalf("a cache hit still called the provider: %d calls", got)
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}
	notModified := getEntryAudio(t, env.e, env.session, entryID, etag)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("conditional audio: %d %q", notModified.Code, notModified.Body.String())
	}

	// Case and repeated spaces change the stored display text but not the spoken text key.
	sameSpeech := createEntry(t, env.e, env.session, topicID, "  "+strings.ToUpper(text)+"  ")
	third := getEntryAudio(t, env.e, env.session, sameSpeech, "")
	if third.Code != http.StatusOK || third.Body.String() != string(tts.body) {
		t.Fatalf("normalized cache key: %d %q", third.Code, third.Body.String())
	}
	if got := len(tts.requestQueries()); got != 1 {
		t.Fatalf("a normalized duplicate called the provider: %d calls", got)
	}
}

func TestEntryAudioUsesScopedTokensAndOwnerChecks(t *testing.T) {
	env, tts, _ := setupAudioEnv(t)
	topicID := topic(t, env.e, env.session)
	entryID := createEntry(t, env.e, env.session, topicID, "scoped audio "+uuid.NewString())

	_, readWrite := apiToken(t, env.e, env.session, "read_write", topicID)
	_, captureOnly := apiToken(t, env.e, env.session, "capture", topicID)
	stranger := token.NewCodec(authSecret).Issue(uuid.New(), time.Now().UTC(), time.Hour)

	for name, tc := range map[string]struct {
		raw  string
		want int
	}{
		"owner":            {env.session, http.StatusOK},
		"read-write token": {readWrite, http.StatusOK},
		"capture token":    {captureOnly, http.StatusForbidden},
		"stranger":         {stranger, http.StatusNotFound},
	} {
		rec := getEntryAudio(t, env.e, tc.raw, entryID, "")
		if rec.Code != tc.want {
			t.Fatalf("%s: %d %q", name, rec.Code, rec.Body.String())
		}
	}

	if got := len(tts.requestQueries()); got != 1 {
		t.Fatalf("unexpected provider calls: %d", got)
	}
}

func TestEntryAudioFailureIsNotCached(t *testing.T) {
	env, tts, _ := setupAudioEnv(t)
	topicID := topic(t, env.e, env.session)
	entryID := createEntry(t, env.e, env.session, topicID, "retryable audio "+uuid.NewString())

	tts.setStatus(http.StatusBadGateway)
	failed := getEntryAudio(t, env.e, env.session, entryID, "")
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("provider failure: %d %q", failed.Code, failed.Body.String())
	}

	tts.setStatus(http.StatusOK)
	retry := getEntryAudio(t, env.e, env.session, entryID, "")
	if retry.Code != http.StatusOK || retry.Body.String() != string(tts.body) {
		t.Fatalf("retry: %d %q", retry.Code, retry.Body.String())
	}
	if got := len(tts.requestQueries()); got != 2 {
		t.Fatalf("provider calls = %d, want 2", got)
	}
}

func TestEntryAudioCacheIsScopedByLanguage(t *testing.T) {
	env, tts, _ := setupAudioEnv(t)

	code, body := request(t, env.e, "POST", "/api/v1/learning/topics", env.session, map[string]any{"name": "English US", "language": "en-US", "translationLanguage": "ru"}, "")
	if code != http.StatusCreated {
		t.Fatalf("topic: %d %v", code, body)
	}
	englishTopic := data(t, body)["id"].(string)

	code, body = request(t, env.e, "POST", "/api/v1/learning/topics", env.session, map[string]any{"name": "Russian", "language": "ru-RU", "translationLanguage": "en"}, "")
	if code != http.StatusCreated {
		t.Fatalf("topic: %d %v", code, body)
	}
	russianTopic := data(t, body)["id"].(string)

	text := "same displayed text " + uuid.NewString()
	englishEntry := createEntry(t, env.e, env.session, englishTopic, text)
	russianEntry := createEntry(t, env.e, env.session, russianTopic, text)

	if rec := getEntryAudio(t, env.e, env.session, englishEntry, ""); rec.Code != http.StatusOK {
		t.Fatalf("english audio: %d", rec.Code)
	}
	if rec := getEntryAudio(t, env.e, env.session, russianEntry, ""); rec.Code != http.StatusOK {
		t.Fatalf("russian audio: %d", rec.Code)
	}

	queries := tts.requestQueries()
	if len(queries) != 2 {
		t.Fatalf("language-scoped cache called the provider %d times", len(queries))
	}

	first, _ := url.ParseQuery(queries[0])
	second, _ := url.ParseQuery(queries[1])
	if first.Get("tl") != "en" || second.Get("tl") != "ru" {
		t.Fatalf("languages: %q, %q", first.Get("tl"), second.Get("tl"))
	}

	rows, err := env.q.GetLearningAudioCache(context.Background(), db.GetLearningAudioCacheParams{Language: "en", TextHash: learningaudio.TextHash(text)})
	if err != nil || rows.Language != "en" {
		t.Fatalf("english cache row: %+v %v", rows, err)
	}
}
