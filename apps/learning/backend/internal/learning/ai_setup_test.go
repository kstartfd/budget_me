package learning_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// providerStub stands in for the three providers. A GET is the key check; a POST
// is a completion, answered in the shape of whichever provider the path belongs
// to, and remembered so a test can see exactly what the server sent.
type providerStub struct {
	mu    sync.Mutex
	text  string
	calls []providerCall
}

type providerCall struct {
	Path string
	// Key is the credential the server presented, whichever header carried it.
	Key  string
	Body string
}

func (p *providerStub) serve(status int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if r.Method != http.MethodPost {
		_, _ = w.Write([]byte(`{"data":[],"has_more":false,"first_id":null,"last_id":null}`))

		return
	}

	raw, _ := io.ReadAll(r.Body)
	key := r.Header.Get("X-Api-Key")

	if key == "" {
		key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls = append(p.calls, providerCall{Path: r.URL.Path, Key: key, Body: string(raw)})

	switch {
	case strings.HasSuffix(r.URL.Path, "/v1/messages"):
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":` + quote(p.text) + `}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	case strings.HasSuffix(r.URL.Path, "/responses"):
		_, _ = w.Write([]byte(`{"output":[{"type":"reasoning","content":[]},{"type":"message","content":[{"type":"output_text","text":` + quote(p.text) + `}]}]}`))
	default:
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + quote(p.text) + `}}]}`))
	}
}

func (p *providerStub) reply(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.text = text
}

func (p *providerStub) requests() []providerCall {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]providerCall(nil), p.calls...)
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
	// provider records the completions the server asked the providers for. It
	// answers with keyStatus too, so a test can make a provider refuse or fail.
	provider *providerStub
}

func setupAI(t *testing.T) aiEnv {
	t.Helper()

	pool, q := testdb.Pool(t)
	user := testdb.NewUser(t, pool, q, time.Now().UTC(), 1)
	raw := token.NewCodec(authSecret).Issue(user, time.Now().UTC(), time.Hour)

	keyStatus := &atomic.Int32{}
	keyStatus.Store(http.StatusOK)

	provider := &providerStub{}
	providers := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider.serve(int(keyStatus.Load()), w, r)
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
		provider:  provider,
	}
}

func quote(s string) string {
	b, _ := json.Marshal(s)

	return string(b)
}
