package alert

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type spy struct {
	mu   sync.Mutex
	sent []string
}

func (s *spy) Enabled() bool { return true }

func (s *spy) Send(_ context.Context, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent = append(s.sent, text)
}

func (s *spy) messages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.sent...)
}

// waitFor gives the delivery goroutine a moment — Report never blocks the
// response, so the send happens off the caller's thread.
func waitFor(t *testing.T, s *spy, want int) []string {
	t.Helper()

	for range 100 {
		if msgs := s.messages(); len(msgs) >= want {
			return msgs
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("ждали %d сообщений, пришло %d", want, len(s.messages()))

	return nil
}

// A database that went away means one alert per request — the storm must fold
// into a single page plus a summary, or the message that mattered is buried.
func TestReporterFoldsAStormIntoOneAlertAndOneSummary(t *testing.T) {
	s := &spy{}
	r := NewReporter(s, "prod")
	r.window = 60 * time.Millisecond

	defer r.Stop()

	for range 25 {
		r.Report("POST", "/api/v1/transactions", 500, errors.New("connection refused"))
	}

	msgs := waitFor(t, s, 1)
	if len(msgs) != 1 {
		t.Fatalf("первый 5xx должен слать РОВНО один алерт, послано %d", len(msgs))
	}

	if !strings.Contains(msgs[0], "POST /api/v1/transactions 500") ||
		!strings.Contains(msgs[0], "connection refused") {
		t.Errorf("алерт без маршрута/причины: %q", msgs[0])
	}

	// Окно закрылось — приходит сводка о том, что оно проглотило.
	msgs = waitFor(t, s, 2)
	if !strings.Contains(msgs[1], "ещё 24×") {
		t.Errorf("сводка должна назвать 24 подавленных ошибки, получили %q", msgs[1])
	}
}

// Different routes are different incidents: folding them together would hide a
// second failure behind the first.
func TestReporterKeepsRoutesApart(t *testing.T) {
	s := &spy{}
	r := NewReporter(s, "prod")
	r.window = time.Hour

	defer r.Stop()

	r.Report("POST", "/api/v1/transactions", 500, nil)
	r.Report("GET", "/api/v1/insights", 500, nil)

	msgs := waitFor(t, s, 2)
	if len(msgs) != 2 {
		t.Fatalf("два разных маршрута — два алерта, получили %d", len(msgs))
	}
}

// 4xx is the user being wrong, not the server. Alerting on it would train the
// owner to ignore the channel.
func TestReporterIgnoresClientErrors(t *testing.T) {
	s := &spy{}
	r := NewReporter(s, "prod")

	defer r.Stop()

	r.Report("POST", "/api/v1/transactions", 400, nil)
	r.Report("GET", "/api/v1/me", 401, nil)
	r.Report("GET", "/api/v1/plans/:id", 404, nil)

	time.Sleep(30 * time.Millisecond)

	if msgs := s.messages(); len(msgs) != 0 {
		t.Fatalf("4xx не должны алертить, послано %d: %v", len(msgs), msgs)
	}
}

// A box with no chat configured must not fail, log-spam, or panic in a handler.
func TestDisabledClientIsANoOp(t *testing.T) {
	r := NewReporter(New("", ""), "dev")

	defer r.Stop()

	r.Report("POST", "/api/v1/transactions", 500, errors.New("boom"))
}
