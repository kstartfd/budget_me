package commands

import (
	"testing"
	"time"
)

func TestNextReviewClimbsAndFalls(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return now.AddDate(0, 0, n) }

	for name, tc := range map[string]struct {
		stage     int
		grade     string
		wantStage int
		wantDue   time.Time
		wantKnown bool
	}{
		"new, remembered":        {0, GradeGood, 1, day(1), false},
		"second step":            {1, GradeGood, 2, day(2), false},
		"easy skips a stage":     {1, GradeEasy, 3, day(4), false},
		"forgotten starts over":  {5, GradeAgain, 0, now.Add(10 * time.Minute), false},
		"last step":              {6, GradeGood, 7, day(60), false},
		"past the last is known": {7, GradeGood, 7, day(60), true},
		"easy from the top":      {6, GradeEasy, 7, day(60), true},
	} {
		stage, due, known := NextReview(tc.stage, tc.grade, now)
		if stage != tc.wantStage || !due.Equal(tc.wantDue) || known != tc.wantKnown {
			t.Errorf("%s: got stage %d due %v known %v, want %d %v %v", name, stage, due, known, tc.wantStage, tc.wantDue, tc.wantKnown)
		}
	}
}

func TestValidGrade(t *testing.T) {
	if !ValidGrade("good") || ValidGrade("") || ValidGrade("hard") {
		t.Fatal("only again, good and easy are grades")
	}
}
