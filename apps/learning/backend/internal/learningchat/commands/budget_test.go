package commands

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/llm"
)

var (
	profile = llm.ChatProfile{BudgetTokens: 1000, BudgetWindow: 24 * time.Hour}
	noon    = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func spent(tokens int64, since time.Time) db.LearningChat {
	return db.LearningChat{SpentTokens: tokens, SpentSince: pgtype.Timestamptz{Time: since, Valid: true}}
}

func TestBudgetOfAChatNeverSpentIsWhole(t *testing.T) {
	b := BudgetOf(db.LearningChat{}, profile, noon)

	if b.Limit != 1000 || b.Used != 0 || b.ResetsAt != nil || b.Exhausted() {
		t.Fatalf("fresh chat: %+v", b)
	}
}

func TestBudgetCountsWithinTheWindowAndStartsOverAfterIt(t *testing.T) {
	started := noon.Add(-23 * time.Hour)

	b := BudgetOf(spent(900, started), profile, noon)
	if b.Used != 900 || b.Exhausted() || !b.ResetsAt.Equal(started.Add(24*time.Hour)) {
		t.Fatalf("inside the window: %+v", b)
	}

	// The window ends exactly at since+window: from then on the chat is whole.
	b = BudgetOf(spent(5000, started), profile, noon.Add(time.Hour))
	if b.Used != 0 || b.ResetsAt != nil || b.Exhausted() {
		t.Fatalf("window over: %+v", b)
	}
}

func TestBudgetIsExhaustedAtTheLimit(t *testing.T) {
	for used, want := range map[int64]bool{999: false, 1000: true, 1500: true} {
		if got := BudgetOf(spent(used, noon), profile, noon).Exhausted(); got != want {
			t.Errorf("used %d: exhausted %v, want %v", used, got, want)
		}
	}
}

func TestBudgetAfterATurnOpensTheWindowOnlyOnce(t *testing.T) {
	first := BudgetOf(db.LearningChat{}, profile, noon).After(300, noon, profile.BudgetWindow)
	if first.Used != 300 || first.ResetsAt == nil || !first.ResetsAt.Equal(noon.Add(24*time.Hour)) {
		t.Fatalf("first turn: %+v", first)
	}

	later := noon.Add(3 * time.Hour)

	second := first.After(250, later, profile.BudgetWindow)
	if second.Used != 550 || !second.ResetsAt.Equal(noon.Add(24*time.Hour)) {
		t.Fatalf("a later turn must not move the end of the window: %+v", second)
	}
}
