package learningprofile

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

var t0 = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func at(days int, hours ...int) time.Time {
	h := 0
	if len(hours) > 0 {
		h = hours[0]
	}

	return t0.AddDate(0, 0, days).Add(time.Duration(h) * time.Hour)
}

// answer is a right or wrong answer to an exercise of the kind, at the given time.
func answer(kind string, correct bool, when time.Time) Evidence {
	mode, weight, _ := Kind(kind)

	return Evidence{Mode: mode, Correct: correct, Weight: weight, AttemptNo: 1, Assisted: true, At: when}
}

func independent(e Evidence) Evidence {
	e.Assisted = false
	e.Review = true

	return e
}

func TestKindTable(t *testing.T) {
	tests := []struct {
		kind   string
		mode   Mode
		weight float64
	}{
		{"choice", Recognition, 0.4},
		{"listen", Recognition, 0.4},
		{"fill", Recall, 1.0},
		{"cloze", Production, 1.0},
		{"order", Production, 1.0},
		{"speak", Production, 1.0},
		{"translate", Production, 0.7},
		{"write", Production, 0.7},
		{"read", Recall, 0.7},
	}

	for _, tt := range tests {
		mode, weight, ok := Kind(tt.kind)
		if !ok || mode != tt.mode || weight != tt.weight {
			t.Errorf("Kind(%q) = %v, %v, %v; want %v, %v", tt.kind, mode, weight, ok, tt.mode, tt.weight)
		}
	}

	if _, _, ok := Kind("riddle"); ok {
		t.Error("an unknown kind must store no evidence")
	}
}

func TestSkillOf(t *testing.T) {
	tests := []struct{ kind, topic, want string }{
		{"listen", "grammar", SkillListening},
		{"speak", "vocabulary", SkillSpeaking},
		{"read", "grammar", SkillReading},
		{"write", "grammar", SkillWriting},
		{"fill", "vocabulary", SkillVocabulary},
		{"choice", "grammar", SkillGrammar},
		{"fill", "pronunciation", "pronunciation"},
		{"fill", "weird", SkillGrammar},
	}

	for _, tt := range tests {
		if got := SkillOf(tt.kind, tt.topic); got != tt.want {
			t.Errorf("SkillOf(%q, %q) = %q, want %q", tt.kind, tt.topic, got, tt.want)
		}
	}
}

func TestEvidenceWeightRetries(t *testing.T) {
	c := Default()

	tests := []struct {
		base      float64
		attemptNo int
		want      float64
	}{
		{1.0, 1, 1.0},
		{1.0, 2, 0.5},
		{0.7, 3, 0.35},
		{0.4, 2, 0.2}, // half of 0.4 is the floor
		{0.3, 2, 0.2}, // half would be 0.15: the floor holds
		{0.1, 2, 0.1}, // never above what the verdict was worth
	}

	for _, tt := range tests {
		if got := c.EvidenceWeight(tt.base, tt.attemptNo); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("EvidenceWeight(%v, %d) = %v, want %v", tt.base, tt.attemptNo, got, tt.want)
		}
	}
}

func TestFirstAnswerStartsLearningAndSchedulesTomorrow(t *testing.T) {
	c := Default()
	it := c.Apply(Item{}, answer("fill", true, at(0)))

	if it.State != Learning || it.Count != 1 || it.Days != 1 || it.Stage != 1 {
		t.Fatalf("after the first right answer: %+v", it)
	}

	if want := at(1); !it.DueAt.Equal(want) {
		t.Fatalf("due %v, want %v", it.DueAt, want)
	}
}

func TestWrongAnswerSchedulesTomorrowAtTheStart(t *testing.T) {
	c := Default()
	it := c.Apply(Item{}, answer("fill", false, at(0)))

	if it.State != Learning || it.Stage != 0 || !it.DueAt.Equal(at(1)) {
		t.Fatalf("after a wrong first answer: %+v", it)
	}

	if it.RecallSuccesses != 0 || it.LastSuccessAt != (time.Time{}) {
		t.Fatalf("a wrong answer is no success: %+v", it)
	}
}

func TestPracticingEarlyOrRepeatedlyAdvancesNothing(t *testing.T) {
	c := Default()
	it := Item{}

	// Six exercises in one sitting: one step on the ladder, not six.
	for i := range 6 {
		it = c.Apply(it, answer("fill", true, at(0, i)))
	}

	if it.Stage != 1 || !it.DueAt.Equal(at(0, 0).Add(24*time.Hour)) {
		t.Fatalf("six answers in a day: stage %d due %v", it.Stage, it.DueAt)
	}

	if it.Count != 6 || it.Days != 1 {
		t.Fatalf("every answer counts as evidence but one day is one day: %+v", it)
	}
}

func TestRecognitionNeverGoesPastLearning(t *testing.T) {
	c := Default()
	it := Item{}

	for d := range 30 {
		it = c.Apply(it, answer("choice", true, at(d)))
	}

	if it.State != Learning {
		t.Fatalf("thirty days of right picks made the item %s", it.State)
	}

	if it.Stage > 1 {
		t.Fatalf("a pick moved the ladder to stage %d", it.Stage)
	}
}

func TestLearningBecomesPracticedOnASecondDay(t *testing.T) {
	c := Default()
	it := Item{}

	for i := range 3 {
		it = c.Apply(it, answer("fill", true, at(0, i)))
	}

	if it.State != Learning {
		t.Fatalf("three answers on one day: %s, want learning (needs two days)", it.State)
	}

	it = c.Apply(it, answer("fill", true, at(1)))
	if it.State != Practiced {
		t.Fatalf("a fourth right answer on the next day: %s, want practiced (%+v)", it.State, it)
	}
}

func TestPracticedNeedsAProducedAnswerNotJustPicks(t *testing.T) {
	c := Default()
	it := Item{}

	for d := range 4 {
		it = c.Apply(it, answer("choice", true, at(d)))
	}

	if it.State != Learning {
		t.Fatalf("picks only: %s", it.State)
	}

	it = c.Apply(it, answer("fill", true, at(5)))
	if it.State != Practiced {
		t.Fatalf("after a produced answer: %s (%+v)", it.State, it)
	}
}

// reach builds an item that is practiced after a first day of exercises and a
// first review a day later.
func reach(c Config) Item {
	it := Item{}
	for i := range 3 {
		it = c.Apply(it, answer("fill", true, at(0, i)))
	}

	return c.Apply(it, answer("fill", true, at(1)))
}

func TestMasteryNeedsAnIndependentAnswerSevenDaysAfterTheLastSuccess(t *testing.T) {
	c := Default()
	it := reach(c)

	if it.State != Practiced || !it.DueAt.Equal(at(4)) {
		t.Fatalf("setup: %+v", it)
	}

	// Day 4: a right review, but the last success was only three days ago.
	it = c.Apply(it, independent(answer("fill", true, at(4))))
	if it.State != Practiced {
		t.Fatalf("day 4: %s, want practiced", it.State)
	}

	if want := at(4).Add(7 * 24 * time.Hour); !it.DueAt.Equal(want) {
		t.Fatalf("stage %d due %v, want %v", it.Stage, it.DueAt, want)
	}

	// Day 11, but the answer was assisted (the key was in front of the learner).
	assisted := c.Apply(it, answer("fill", true, at(11)))
	if assisted.State != Practiced {
		t.Fatalf("an assisted answer made the item %s", assisted.State)
	}

	// Day 11, independent, seven days after the last success.
	mastered := c.Apply(it, independent(answer("fill", true, at(11))))
	if mastered.State != Mastered {
		t.Fatalf("an independent answer a week later: %s (%+v)", mastered.State, mastered)
	}

	// A recognition answer, even independent, is not enough.
	picked := c.Apply(it, independent(answer("choice", true, at(11))))
	if picked.State != Practiced {
		t.Fatalf("an independent pick made the item %s", picked.State)
	}
}

func TestLapseAndRelearning(t *testing.T) {
	c := Default()
	it := reach(c)
	it = c.Apply(it, independent(answer("fill", true, at(4))))
	it = c.Apply(it, independent(answer("fill", true, at(11))))

	if it.State != Mastered {
		t.Fatalf("setup: %s", it.State)
	}

	lapsed := c.Apply(it, independent(answer("fill", false, at(11, 1))))
	if lapsed.State != Relearning || lapsed.Lapses != 1 || lapsed.Stage != 0 {
		t.Fatalf("after a wrong review: %+v", lapsed)
	}

	if want := at(11, 1).Add(24 * time.Hour); !lapsed.DueAt.Equal(want) {
		t.Fatalf("a lapse comes back tomorrow: %v, want %v", lapsed.DueAt, want)
	}

	// Two right answers in a row on the same day do not bring it back...
	same := c.Apply(lapsed, answer("fill", true, at(11, 2)))
	same = c.Apply(same, answer("fill", true, at(11, 3)))
	if same.State != Relearning {
		t.Fatalf("two on one day: %s, want relearning", same.State)
	}

	// ...on different days they do.
	back := c.Apply(lapsed, answer("fill", true, at(12)))
	if back.State != Relearning {
		t.Fatalf("one right answer: %s", back.State)
	}

	back = c.Apply(back, answer("fill", true, at(13)))
	if back.State != Practiced {
		t.Fatalf("two on different days: %s (%+v)", back.State, back)
	}
}

func TestWrongAnswerBeforeItWasLearntIsNoLapse(t *testing.T) {
	c := Default()
	it := c.Apply(Item{}, answer("fill", true, at(0)))
	it = c.Apply(it, answer("fill", false, at(1)))

	if it.State != Learning || it.Lapses != 0 {
		t.Fatalf("a wrong answer to a learning item: %+v", it)
	}
}

func TestRetryCountsHalf(t *testing.T) {
	c := Default()
	first := answer("fill", false, at(0))
	retry := answer("fill", true, at(0)) // the same instant: nothing has faded yet
	retry.AttemptNo = 2

	it := c.Apply(c.Apply(Item{}, first), retry)

	if math.Abs(it.WeightSum-1.5) > 1e-9 || math.Abs(it.WeightedSum-0.5) > 1e-9 {
		t.Fatalf("weights %v / %v, want 1.5 / 0.5", it.WeightSum, it.WeightedSum)
	}
}

func TestOldEvidenceFades(t *testing.T) {
	c := Default()
	it := c.Apply(Item{}, answer("fill", false, at(0)))
	it = c.Apply(it, answer("fill", true, at(60)))

	// Two half-lives later the old miss weighs a quarter.
	if want := 1.0 / (1.0 + 0.25); math.Abs(it.Score()-want) > 1e-9 {
		t.Fatalf("score %v, want %v", it.Score(), want)
	}
}

func TestReviewState(t *testing.T) {
	c := Default()
	now := at(10)

	tests := []struct {
		name string
		due  time.Time
		want ReviewState
	}{
		{"never scheduled", time.Time{}, NotScheduled},
		{"in the future", now.Add(time.Hour), Scheduled},
		{"just due", now.Add(-time.Hour), Due},
		{"a day late", now.Add(-24 * time.Hour), Overdue},
		{"long overdue", now.Add(-10 * 24 * time.Hour), Overdue},
	}

	for _, tt := range tests {
		if got := c.ReviewOf(tt.due, now); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestPublicStateNames(t *testing.T) {
	tests := []struct {
		in     State
		name   string
		lapsed bool
	}{
		{Unseen, "not_started", false},
		{Introduced, "learning", false},
		{Learning, "learning", false},
		{Relearning, "learning", true},
		{Practiced, "practiced", false},
		{Mastered, "mastered", false},
	}

	for _, tt := range tests {
		if name, lapsed := Public(tt.in); name != tt.name || lapsed != tt.lapsed {
			t.Errorf("Public(%s) = %s, %v; want %s, %v", tt.in, name, lapsed, tt.name, tt.lapsed)
		}
	}
}

// TestInvariantsOverRandomHistories runs many seeded random histories and checks
// what must hold whatever the learner does.
func TestInvariantsOverRandomHistories(t *testing.T) {
	c := Default()
	kindsList := []string{"choice", "listen", "fill", "cloze", "order", "speak", "translate", "write", "read"}
	rng := rand.New(rand.NewSource(7))

	for run := range 300 {
		it := Item{}
		now := t0
		var seen bool

		for range 40 {
			now = now.Add(time.Duration(rng.Intn(72*60)) * time.Minute)
			ev := answer(kindsList[rng.Intn(len(kindsList))], rng.Intn(100) < 65, now)
			ev.Assisted = rng.Intn(2) == 0
			ev.Review = !ev.Assisted
			ev.AttemptNo = 1 + rng.Intn(3)

			before := it
			it = c.Apply(it, ev)

			if s := it.Score(); s < 0 || s > 1 || math.IsNaN(s) {
				t.Fatalf("run %d: score %v", run, s)
			}

			if it.Stage < 0 || it.Stage > len(c.Ladder) {
				t.Fatalf("run %d: stage %d out of 0..%d", run, it.Stage, len(c.Ladder))
			}

			if !it.DueAt.After(ev.At) {
				t.Fatalf("run %d: due %v is not after the answer at %v", run, it.DueAt, ev.At)
			}

			if it.Count != before.Count+1 {
				t.Fatalf("run %d: count %d -> %d", run, before.Count, it.Count)
			}

			if it.Lapses < before.Lapses {
				t.Fatalf("run %d: lapses went down", run)
			}

			if it.State == "" || it.State == Unseen {
				t.Fatalf("run %d: state %q after an answer", run, it.State)
			}

			if it.State == Mastered && it.RecallSuccesses == 0 {
				t.Fatalf("run %d: mastered without a produced answer", run)
			}

			if it.State == Practiced || it.State == Mastered {
				seen = true
			}
		}

		_ = seen
	}
}
