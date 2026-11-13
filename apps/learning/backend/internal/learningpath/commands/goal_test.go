package commands

import (
	"strings"
	"testing"
)

func TestTheWeeklyGoalIsReadFromWhatWasStoredWithTheProgram(t *testing.T) {
	for _, tc := range []struct {
		stored string
		want   int
	}{
		{`{}`, 0},
		{``, 0},
		{`not json`, 0},
		{`{"hoursPerWeek":5}`, 300},
		{`{"hoursPerWeek":40}`, 2400},
		{`{"hoursPerWeek":41}`, 0},
		{`{"hoursPerWeek":-2}`, 0},
		// The survey used to ask minutes a day and days a week.
		{`{"minutesPerDay":30,"daysPerWeek":5}`, 150},
		{`{"minutesPerDay":30}`, 0},
		{`{"daysPerWeek":5}`, 0},
		// The new answer wins over the old ones.
		{`{"hoursPerWeek":2,"minutesPerDay":60,"daysPerWeek":7}`, 120},
	} {
		if got := WeeklyGoalMinutes([]byte(tc.stored)); got != tc.want {
			t.Errorf("%q: goal %d, want %d", tc.stored, got, tc.want)
		}
	}
}

func TestHoursPerWeekOutOfRangeAreDroppedFromThePreferences(t *testing.T) {
	for hours, want := range map[int]int{-1: 0, 0: 0, 1: 1, 10: 10, MaxHoursPerWeek: MaxHoursPerWeek, MaxHoursPerWeek + 1: 0} {
		if got := (Preferences{HoursPerWeek: hours}).Clean().HoursPerWeek; got != want {
			t.Errorf("%d hours a week cleaned to %d, want %d", hours, got, want)
		}
	}
}

// TestThePlannersAreToldTheHoursAWeek: both prompts that size a program to the
// learner's time speak of hours a week, not of days.
func TestThePlannersAreToldTheHoursAWeek(t *testing.T) {
	for name, system := range map[string]string{"analyze": analyzeSystem, "plan": planSystem} {
		if !strings.Contains(system, "hoursPerWeek") || strings.Contains(system, "days a week") {
			t.Errorf("%s does not size the program by hours a week", name)
		}
	}
}
