package dayemail

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"budget_me/backend/internal/db"
)

func TestDueSlotUsesLocalHoursAndFrequency(t *testing.T) {
	zone := time.FixedZone("test", 3*60*60)
	for _, tc := range []struct {
		hour      int
		frequency db.DayEmailFrequency
		slot      db.DayEmailSlot
		due       bool
	}{
		{8, db.DayEmailFrequencyTwice, "", false},
		{9, db.DayEmailFrequencyDaily, db.DayEmailSlotMorning, true},
		{11, db.DayEmailFrequencyDaily, db.DayEmailSlotMorning, true},
		{18, db.DayEmailFrequencyDaily, "", false},
		{18, db.DayEmailFrequencyTwice, db.DayEmailSlotEvening, true},
		{21, db.DayEmailFrequencyTwice, "", false},
	} {
		at := time.Date(2026, 9, 25, tc.hour, 0, 0, 0, zone)
		slot, due := dueSlot(at, tc.frequency)
		if slot != tc.slot || due != tc.due {
			t.Errorf("hour=%d frequency=%s: slot=%s due=%v", tc.hour, tc.frequency, slot, due)
		}
	}
}

func TestDigestTextMarksOverdueAndLimitsLength(t *testing.T) {
	tasks := make([]db.ListOpenTasksForDigestRow, 32)
	for i := range tasks {
		tasks[i] = db.ListOpenTasksForDigestRow{
			Title: "Task", PlannedFor: pgtype.Date{Time: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), Valid: true},
		}
	}
	subject, body := digestText(tasks, "2026-09-25", db.DayEmailSlotMorning, nil, "https://budgetme.xyz")
	if subject == "" || strings.Count(body, "• Task") != 30 || !strings.Contains(body, "Осталось ещё задач: 2") || !strings.Contains(body, "с прошлого дня") {
		t.Fatalf("unexpected digest: subject=%q body=%q", subject, body)
	}
}
