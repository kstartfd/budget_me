package learning_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

// fakeLLM answers the capture queue's agent calls and remembers what it was asked.
type fakeLLM struct {
	mu    sync.Mutex
	reply string
	err   error
	calls []llm.Request
}

func (f *fakeLLM) Complete(_ context.Context, req llm.Request) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, req)

	return f.reply, f.err
}

func (f *fakeLLM) set(reply string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reply, f.err = reply, err
}

func (f *fakeLLM) requests() []llm.Request {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]llm.Request(nil), f.calls...)
}

type aiEnv struct {
	e         *echo.Echo
	q         *db.Queries
	pool      *pgxpool.Pool
	user      uuid.UUID
	session   string
	processor commands.ProcessHandler
	agent     *fakeLLM
	// keyStatus is what the stand-in providers answer when the settings
	// endpoint checks a key: 200 accepts it, 401 rejects it.
	keyStatus *atomic.Int32
}

func setupAI(t *testing.T) aiEnv {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)
	raw := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	keyStatus := &atomic.Int32{}
	keyStatus.Store(http.StatusOK)

	providers := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(keyStatus.Load()))
		_, _ = w.Write([]byte(`{"data":[],"has_more":false,"first_id":null,"last_id":null}`))
	}))
	t.Cleanup(providers.Close)

	cfg := config.Config{AuthSecret: authSecret, LLMURLs: llm.URLs{Anthropic: providers.URL, OpenAI: providers.URL, DeepSeek: providers.URL}}
	agent := &fakeLLM{}

	return aiEnv{
		e:         server.New(cfg, pool, testdb.AccountPool(t)),
		q:         q,
		pool:      pool,
		user:      user,
		session:   raw,
		processor: commands.NewProcessHandler(pool, clock.New(), secretbox.New(authSecret), agent),
		agent:     agent,
		keyStatus: keyStatus,
	}
}
