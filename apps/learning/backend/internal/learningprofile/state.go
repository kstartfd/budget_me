package learningprofile

import (
	"math"
	"time"
)

// State of one piece of knowledge for one learner. No row at all means the
// learner has shown nothing yet: unknown, which is not the same as not started.
type State string

const (
	Unseen     State = "unseen"
	Introduced State = "introduced"
	Learning   State = "learning"
	Practiced  State = "practiced"
	Mastered   State = "mastered"
	Relearning State = "relearning"
)

// Public is how the API names a state (decision D-028): the internal relearning
// is a learning item that has lapsed, and an item with no evidence is not started.
func Public(s State) (name string, lapsed bool) {
	switch s {
	case Practiced, Mastered:
		return string(s), false
	case Relearning:
		return "learning", true
	case Learning, Introduced:
		return "learning", false
	}

	return "not_started", false
}

// Config holds every number of the rules. Default is the documented starting
// point; a test or a later tuning passes its own.
type Config struct {
	// Ladder is the wait after a spaced success: the first success schedules
	// Ladder[0], the second Ladder[1], and the last rung repeats.
	Ladder []time.Duration
	// HalfLife is how fast old evidence fades from the recent score.
	HalfLife time.Duration

	PracticedScore      float64
	PracticedEvidence   int
	PracticedDays       int
	MasteredScore       float64
	MasteredGap         time.Duration
	RelearnStreak       int
	RetryFactor         float64
	WeightFloor         float64
	OverdueAfter        time.Duration
	RecognitionMaxStage int
	FailureWait         time.Duration
}

const day = 24 * time.Hour

// Default is the documented starting configuration (D-029: a ladder of 1, 3, 7,
// 14 and 30 days).
func Default() Config {
	return Config{
		Ladder:              []time.Duration{1 * day, 3 * day, 7 * day, 14 * day, 30 * day},
		HalfLife:            30 * day,
		PracticedScore:      0.8,
		PracticedEvidence:   3,
		PracticedDays:       2,
		MasteredScore:       0.85,
		MasteredGap:         7 * day,
		RelearnStreak:       2,
		RetryFactor:         0.5,
		WeightFloor:         0.2,
		OverdueAfter:        day,
		RecognitionMaxStage: 1,
		FailureWait:         1 * day,
	}
}

// Evidence is one answer as the rules see it.
type Evidence struct {
	Mode    Mode
	Correct bool
	// Weight is the reliability of the verdict (Kind), before the retry rule.
	Weight float64
	// AttemptNo is which try of the exercise this was, 1 for the first.
	AttemptNo int
	// Assisted: the key or a hint was in front of the learner. An assisted answer
	// can build an item up but cannot make it mastered.
	Assisted bool
	// Review: the answer was given to a review that was due.
	Review bool
	At     time.Time
}

// Item is what is known about one piece of knowledge for one learner. The zero
// value is "nothing shown yet".
type Item struct {
	State           State
	WeightedSum     float64
	WeightSum       float64
	Count           int
	Days            int
	RecallSuccesses int
	Streak          int
	StreakDay       time.Time
	Stage           int
	Lapses          int
	LastEvidenceAt  time.Time
	LastSuccessAt   time.Time
	DueAt           time.Time
}

// Score is the recency-weighted share of right answers, 0 to 1. It is an
// estimate of how reliably the learner answers, not the chance they know it.
func (it Item) Score() float64 {
	if it.WeightSum <= 0 {
		return 0
	}

	return math.Min(1, math.Max(0, it.WeightedSum/it.WeightSum))
}

// EvidenceWeight is the weight an answer carries: the verdict's reliability, halved
// for a retry (a second try after seeing the right answer proves less).
func (c Config) EvidenceWeight(base float64, attemptNo int) float64 {
	if attemptNo <= 1 {
		return base
	}

	return math.Max(base*c.RetryFactor, math.Min(base, c.WeightFloor))
}

// utcDay is the UTC calendar day of t, as an instant.
func utcDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Apply is the whole rule: it folds one answer into what is known about the item.
// Evidence must arrive in time order for one item; the caller holds a row lock.
func (c Config) Apply(it Item, ev Evidence) Item {
	prevSuccess := it.LastSuccessAt
	wasDue := !it.DueAt.IsZero() && !ev.At.Before(it.DueAt)
	weight := c.EvidenceWeight(ev.Weight, ev.AttemptNo)
	correct := ev.Correct
	recall := correct && ev.Mode != Recognition

	// Recent score: older evidence fades with the half-life, then this answer is added.
	if !it.LastEvidenceAt.IsZero() && ev.At.After(it.LastEvidenceAt) {
		fade := math.Pow(0.5, float64(ev.At.Sub(it.LastEvidenceAt))/float64(c.HalfLife))
		it.WeightedSum *= fade
		it.WeightSum *= fade
	}

	it.WeightSum += weight
	if correct {
		it.WeightedSum += weight
	}

	it.Count++

	if it.LastEvidenceAt.IsZero() || !utcDay(ev.At).Equal(utcDay(it.LastEvidenceAt)) {
		it.Days++
	}

	if recall {
		it.RecallSuccesses++
		if it.Streak == 0 {
			it.StreakDay = utcDay(ev.At)
		}

		it.Streak++
	} else if !correct {
		it.Streak = 0
		it.StreakDay = time.Time{}
	}

	if it.State == "" || it.State == Unseen || it.State == Introduced {
		it.State = Learning
	}

	// A wrong answer to something that was learnt, when it was due, is a lapse.
	lapse := !correct && (ev.Review || wasDue) && (it.State == Practiced || it.State == Mastered)
	if lapse {
		it.State = Relearning
		it.Lapses++
	}

	c.schedule(&it, ev, correct, wasDue)

	if correct {
		it.LastSuccessAt = ev.At
	}

	it.LastEvidenceAt = ev.At
	c.promote(&it, ev, recall, prevSuccess)

	return it
}

// schedule moves the item on the review ladder. A success advances it only when
// the item was due, so practising early, or answering six exercises in a row,
// advances nothing; a failure sends the item back to the start of the ladder.
func (c Config) schedule(it *Item, ev Evidence, correct, wasDue bool) {
	if !correct {
		it.Stage = 0
		it.DueAt = ev.At.Add(c.FailureWait)

		return
	}

	switch {
	case it.DueAt.IsZero():
		it.Stage = 1
	case wasDue && ev.Mode == Recognition:
		// A pick proves too little to move an item up the ladder.
		it.Stage = max(it.Stage, min(1, c.RecognitionMaxStage))
	case wasDue:
		it.Stage++
	default:
		return
	}

	it.Stage = min(it.Stage, len(c.Ladder))
	it.DueAt = ev.At.Add(c.Ladder[max(it.Stage, 1)-1])
}

// promote moves the item up when the evidence now justifies it.
func (c Config) promote(it *Item, ev Evidence, recall bool, prevSuccess time.Time) {
	score := it.Score()

	switch it.State {
	case Learning:
		if score >= c.PracticedScore && it.Count >= c.PracticedEvidence && it.RecallSuccesses >= 1 && it.Days >= c.PracticedDays {
			it.State = Practiced
		}
	case Practiced:
		independent := !ev.Assisted
		retained := !prevSuccess.IsZero() && ev.At.Sub(prevSuccess) >= c.MasteredGap

		if recall && independent && retained && score >= c.MasteredScore {
			it.State = Mastered
		}
	case Relearning:
		if it.Streak >= c.RelearnStreak && !it.StreakDay.IsZero() && !utcDay(ev.At).Equal(it.StreakDay) {
			it.State = Practiced
		}
	}
}

// ReviewState is when an item is to be asked again, read from its due time.
type ReviewState string

const (
	NotScheduled ReviewState = "not_scheduled"
	Scheduled    ReviewState = "scheduled"
	Due          ReviewState = "due"
	Overdue      ReviewState = "overdue"
)

// ReviewOf is derived at read time and never stored, so it cannot go stale.
func (c Config) ReviewOf(due, now time.Time) ReviewState {
	switch {
	case due.IsZero():
		return NotScheduled
	case due.After(now):
		return Scheduled
	case now.Sub(due) >= c.OverdueAfter:
		return Overdue
	}

	return Due
}
