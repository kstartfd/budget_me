package queries

import "testing"

// The one call the model must never make itself: whether a limit is close. It
// used to get bare "потрачено 40.00, лимит 100.00" pairs while being told not to
// do arithmetic, and wrote «кофе близко к лимиту» at 40% spent.
func TestCategoryFactVerdict(t *testing.T) {
	tests := []struct {
		name string
		fact CategoryFact
		want string
	}{
		{
			name: "well within the limit",
			fact: CategoryFact{SpentMinor: 4000, LimitMinor: 10000},
			want: "в пределах лимита, осталось 60.00",
		},
		{
			name: "at the near-limit threshold",
			fact: CategoryFact{SpentMinor: 17270, LimitMinor: 20000},
			want: "БЛИЗКО К ЛИМИТУ, осталось 27.30",
		},
		{
			name: "over the limit",
			fact: CategoryFact{SpentMinor: 15000, LimitMinor: 10000},
			want: "ЛИМИТ ПРЕВЫШЕН на 50.00",
		},
		{
			name: "limit set, nothing spent yet",
			fact: CategoryFact{SpentMinor: 0, LimitMinor: 20000},
			want: "в пределах лимита, осталось 200.00",
		},
		{
			name: "no limit at all",
			fact: CategoryFact{SpentMinor: 3000},
			want: "лимита нет",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fact.Verdict(); got != tt.want {
				t.Errorf("Verdict() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A prompt change must invalidate cached advice, or the nightly pass keeps
// serving text written from the old one.
func TestHashCarriesPromptVersion(t *testing.T) {
	facts := Facts{Month: "2026-07", SpentMinor: 4000}

	if facts.Hash() == "" {
		t.Fatal("Hash() is empty")
	}

	bumped := facts
	bumped.Untouched = []string{"Кофе"}

	if facts.Hash() == bumped.Hash() {
		t.Error("Hash() ignores the category list the advice was written from")
	}
}
