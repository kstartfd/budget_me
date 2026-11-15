package commands

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestWeekAndMonthAreCalendar(t *testing.T) {
	// 2026-11-15 is a Sunday: its week is Mon 9 to Sun 15, the month Nov 1-30.
	ws, we := WeekBounds(date(2026, 11, 15))
	if !ws.Equal(date(2026, 11, 9)) || !we.Equal(date(2026, 11, 15)) {
		t.Fatalf("week of a Sunday: %v - %v", ws, we)
	}

	ws, we = WeekBounds(date(2026, 11, 16)) // a Monday starts a new week
	if !ws.Equal(date(2026, 11, 16)) || !we.Equal(date(2026, 11, 22)) {
		t.Fatalf("week of a Monday: %v - %v", ws, we)
	}

	ms, me := MonthBounds(date(2026, 2, 10))
	if !ms.Equal(date(2026, 2, 1)) || !me.Equal(date(2026, 2, 28)) {
		t.Fatalf("february: %v - %v", ms, me)
	}
}

func TestAccrueCarriesMissedDaysUpToTheCap(t *testing.T) {
	d := date(2026, 11, 10)

	for name, tc := range map[string]struct{ credit, n, days, want int }{
		"same day adds nothing": {0, 10, 0, 0},
		"one day":               {0, 10, 1, 10},
		"three days pile up":    {0, 10, 3, 30},
		"leftover stays":        {4, 10, 2, 24},
		"capped":                {0, 50, 30, maxCredit},
	} {
		if got := Accrue(tc.credit, tc.n, d, d.AddDate(0, 0, tc.days)); got != tc.want {
			t.Errorf("%s: got %d, want %d", name, got, tc.want)
		}
	}
}

func TestLocalDateFollowsTheLearnersZone(t *testing.T) {
	now := time.Date(2026, 11, 15, 22, 30, 0, 0, time.UTC) // already the 16th in Minsk (UTC+3)

	got, err := LocalDate(now, "Europe/Minsk")
	if err != nil || !got.Equal(date(2026, 11, 16)) {
		t.Fatalf("Minsk: %v %v", got, err)
	}

	if got, _ := LocalDate(now, "UTC"); !got.Equal(date(2026, 11, 15)) {
		t.Fatalf("UTC: %v", got)
	}

	if _, err := LocalDate(now, "Not/AZone"); err == nil {
		t.Fatal("an unknown zone must be refused")
	}
}
