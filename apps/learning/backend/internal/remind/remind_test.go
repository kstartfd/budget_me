package remind

import (
	"strings"
	"testing"
	"time"
)

func TestDueWaitsForTheLearnersMorningAndSendsOnceADay(t *testing.T) {
	// 06:30 UTC is 09:30 in Minsk (UTC+3), 07:30 in Berlin in November.
	now := time.Date(2026, 11, 15, 6, 30, 0, 0, time.UTC)

	date, ok := Due(now, "Europe/Minsk", nil)
	if !ok || !date.Equal(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("morning in Minsk: got %v %v", date, ok)
	}

	if _, ok := Due(now, "Europe/Berlin", nil); ok {
		t.Fatal("it is still before 9 in Berlin")
	}

	if _, ok := Due(now, "Europe/Minsk", &date); ok {
		t.Fatal("already sent on that local date")
	}

	yesterday := date.AddDate(0, 0, -1)
	if _, ok := Due(now, "Europe/Minsk", &yesterday); !ok {
		t.Fatal("yesterday's send must not block today's")
	}

	late := time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC) // 23:00 in Minsk
	if _, ok := Due(late, "Europe/Minsk", nil); ok {
		t.Fatal("no reminders late at night")
	}

	if _, ok := Due(now, "Not/AZone", nil); ok {
		t.Fatal("an unknown zone is skipped")
	}
}

func TestMessageSpeaksTheLearnersLanguage(t *testing.T) {
	for lang, tc := range map[string]struct {
		n    int
		want string
	}{
		"ru": {1, "1 карточка"}, "ru-RU": {3, "3 карточки"}, "ru ": {11, "11 карточек"},
		"es": {5, "5 tarjetas"}, "en": {1, "1 card"}, "": {7, "7 cards"}, "de": {2, "2 cards"},
	} {
		text, button := Message(strings.TrimSpace(lang), tc.n)
		if !strings.Contains(text, tc.want) || button == "" {
			t.Errorf("%q: got %q / %q, want %q", lang, text, button, tc.want)
		}
	}

	if text, _ := Message("ru", 21); !strings.Contains(text, "21 карточка") {
		t.Errorf("21 takes the singular form: %q", text)
	}
}
