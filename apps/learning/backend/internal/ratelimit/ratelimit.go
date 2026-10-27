// Package ratelimit keeps one caller from making too many requests: a token
// bucket for how fast, and a cap on how many may be running at once.
package ratelimit

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Config is how fast one caller may go. The zero value means "use the defaults
// of whoever reads it".
type Config struct {
	// Burst is how many requests may arrive back to back.
	Burst int
	// Every is how long one spent request takes to come back.
	Every time.Duration
	// InFlight is how many requests of one caller may run at once.
	InFlight int
}

// sweepAbove is the number of tracked callers past which idle ones are dropped,
// so the map cannot grow with every user that ever asked.
const sweepAbove = 1024

type bucket struct {
	tokens   float64
	last     time.Time
	inFlight int
}

// Limiter is safe for concurrent use. It lives in one process: with several
// running, each keeps its own count, which only ever makes the limit looser by
// the number of processes.
type Limiter struct {
	cfg Config
	now func() time.Time

	mu      sync.Mutex
	callers map[uuid.UUID]*bucket
}

func New(cfg Config) *Limiter {
	return &Limiter{cfg: cfg, now: time.Now, callers: map[uuid.UUID]*bucket{}}
}

// refill brings a bucket up to date at `now`.
func (l *Limiter) refill(b *bucket, now time.Time) {
	if gained := float64(now.Sub(b.last)) / float64(l.cfg.Every); gained > 0 {
		b.tokens = min(float64(l.cfg.Burst), b.tokens+gained)
	}

	b.last = now
}

// Acquire admits one request of `caller` or says how long to wait. On success
// the returned release must be called when the request is finished.
func (l *Limiter) Acquire(caller uuid.UUID) (release func(), retryAfter time.Duration, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	if len(l.callers) > sweepAbove {
		l.sweep(now)
	}

	b, found := l.callers[caller]
	if !found {
		b = &bucket{tokens: float64(l.cfg.Burst), last: now}
		l.callers[caller] = b
	}

	l.refill(b, now)

	switch {
	case b.inFlight >= l.cfg.InFlight:
		// Nothing says when a running request ends; one refill period is a fair guess.
		return nil, l.cfg.Every, false
	case b.tokens < 1:
		return nil, time.Duration((1 - b.tokens) * float64(l.cfg.Every)), false
	}

	b.tokens--
	b.inFlight++

	var once sync.Once

	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()

			b.inFlight--
		})
	}, 0, true
}

// sweep forgets callers that are idle and whose bucket is full again: they are
// indistinguishable from callers never seen.
func (l *Limiter) sweep(now time.Time) {
	for id, b := range l.callers {
		l.refill(b, now)

		if b.inFlight == 0 && b.tokens >= float64(l.cfg.Burst) {
			delete(l.callers, id)
		}
	}
}
