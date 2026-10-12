package alert

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// window is how long one route's 5xx storm is folded into a single thread: the
// first failure pages immediately (you want to know NOW), the rest are counted
// and land as one summary when it closes. Without this a database that went away
// would send an alert per request and bury the one message that mattered.
const window = 10 * time.Minute

// maxDetail bounds the error text carried into the chat. It is OUR message (the
// handler's error), never a request body — the bodies in this app are somebody's
// money.
const maxDetail = 200

// Sender is the delivery half, so the aggregation can be tested without talking
// to Telegram.
type Sender interface {
	Send(ctx context.Context, text string)
	Enabled() bool
}

// Reporter folds 5xx responses into alerts. Safe for concurrent handlers.
type Reporter struct {
	sender Sender
	env    string
	window time.Duration

	mu     sync.Mutex
	storms map[string]*storm
}

// storm is one route's open window: how many failures it has swallowed since the
// alert that opened it.
type storm struct {
	suppressed int
	timer      *time.Timer
}

// NewReporter builds the 5xx reporter. A disabled sender makes every Report a
// no-op — dev boxes stay quiet and nothing in a handler depends on alerting.
func NewReporter(sender Sender, env string) *Reporter {
	return &Reporter{
		sender: sender,
		env:    env,
		window: window,
		storms: make(map[string]*storm),
	}
}

// Report is called from the HTTP error sink for every 5xx. It never blocks the
// response: delivery happens on its own goroutine, and the only work done under
// the lock is counting.
func (r *Reporter) Report(method, route string, status int, err error) {
	if r == nil || !r.sender.Enabled() || status < 500 {
		return
	}

	// The ROUTE, not the URL: "/api/v1/transactions/:id" instead of a path with
	// somebody's transaction id in it.
	key := fmt.Sprintf("%s %s %d", method, route, status)

	r.mu.Lock()

	if open, ok := r.storms[key]; ok {
		open.suppressed++
		r.mu.Unlock()

		return
	}

	s := &storm{}
	r.storms[key] = s
	s.timer = time.AfterFunc(r.window, func() { r.close(key) })

	r.mu.Unlock()

	go r.sender.Send(context.Background(), r.message(key, err))
}

// close ends a window: report what it swallowed, if anything, and let the route
// alert again.
func (r *Reporter) close(key string) {
	r.mu.Lock()

	s, ok := r.storms[key]
	if !ok {
		r.mu.Unlock()

		return
	}

	delete(r.storms, key)

	suppressed := s.suppressed

	r.mu.Unlock()

	if suppressed == 0 {
		return
	}

	r.sender.Send(context.Background(), fmt.Sprintf(
		"🔴 budget_me [%s]: ещё %d× %s за %s",
		r.env, suppressed, key, r.window,
	))
}

func (r *Reporter) message(key string, err error) string {
	var b strings.Builder

	fmt.Fprintf(&b, "🔴 budget_me [%s]: %s", r.env, key)

	if err != nil {
		detail := err.Error()
		if len(detail) > maxDetail {
			detail = detail[:maxDetail] + "…"
		}

		fmt.Fprintf(&b, "\n%s", detail)
	}

	return b.String()
}

// Stop releases the pending window timers. Only the tests need it; prod runs
// until the process dies.
func (r *Reporter) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, s := range r.storms {
		s.timer.Stop()
		delete(r.storms, key)
	}
}
