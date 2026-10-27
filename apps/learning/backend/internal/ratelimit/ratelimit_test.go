package ratelimit

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func limiter(cfg Config) (*Limiter, *time.Time) {
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l := New(cfg)
	l.now = func() time.Time { return at }

	return l, &at
}

func TestBurstThenOneEveryPeriod(t *testing.T) {
	l, at := limiter(Config{Burst: 3, Every: 4 * time.Second, InFlight: 10})
	user := uuid.New()

	for i := 0; i < 3; i++ {
		release, _, ok := l.Acquire(user)
		if !ok {
			t.Fatalf("request %d inside the burst was refused", i+1)
		}

		release()
	}

	_, wait, ok := l.Acquire(user)
	if ok || wait != 4*time.Second {
		t.Fatalf("a fourth request in the same instant: ok %v, wait %v", ok, wait)
	}

	*at = at.Add(3 * time.Second)

	if _, wait, ok = l.Acquire(user); ok || wait != time.Second {
		t.Fatalf("a second short of the next token: ok %v, wait %v", ok, wait)
	}

	*at = at.Add(time.Second)

	if release, _, ok := l.Acquire(user); !ok {
		t.Fatal("one token should have come back after a full period")
	} else {
		release()
	}
}

func TestIdleTimeNeverBuildsUpMoreThanTheBurst(t *testing.T) {
	l, at := limiter(Config{Burst: 2, Every: time.Second, InFlight: 10})
	user := uuid.New()

	*at = at.Add(time.Hour)

	for i := 0; i < 2; i++ {
		release, _, ok := l.Acquire(user)
		if !ok {
			t.Fatalf("request %d refused", i+1)
		}

		release()
	}

	if _, _, ok := l.Acquire(user); ok {
		t.Fatal("an hour of rest must not buy more than the burst")
	}
}

func TestInFlightCapIsPerCallerAndReleaseFreesIt(t *testing.T) {
	l, _ := limiter(Config{Burst: 100, Every: time.Second, InFlight: 2})
	user, other := uuid.New(), uuid.New()

	first, _, _ := l.Acquire(user)
	_, _, _ = l.Acquire(user)

	if _, _, ok := l.Acquire(user); ok {
		t.Fatal("a third running request must be refused")
	}

	if release, _, ok := l.Acquire(other); !ok {
		t.Fatal("another caller is not affected")
	} else {
		release()
	}

	first()
	first() // releasing twice must not free a second slot

	if _, _, ok := l.Acquire(user); !ok {
		t.Fatal("a finished request frees its slot")
	}

	if _, _, ok := l.Acquire(user); ok {
		t.Fatal("a double release must not free more than one slot")
	}
}

func TestIdleCallersAreForgotten(t *testing.T) {
	l, at := limiter(Config{Burst: 1, Every: time.Second, InFlight: 1})

	for i := 0; i <= sweepAbove; i++ {
		release, _, _ := l.Acquire(uuid.New())
		release()
	}

	*at = at.Add(time.Minute)

	if _, _, ok := l.Acquire(uuid.New()); !ok {
		t.Fatal("a newcomer must be admitted")
	}

	if len(l.callers) > 2 {
		t.Fatalf("idle callers were kept: %d", len(l.callers))
	}
}
