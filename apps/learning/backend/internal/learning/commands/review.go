package commands

import (
	"context"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// Grades of a review, as the learner presses them.
const (
	GradeAgain = "again" // forgot it
	GradeGood  = "good"  // remembered
	GradeEasy  = "easy"  // knew it at once
)

// reviewDays is how long an entry rests after each stage it has reached: a
// Leitner ladder, doubling and then some. The step past the last one makes the
// entry known.
var reviewDays = []int{1, 2, 4, 7, 15, 30, 60}

// againAfter is when a forgotten entry comes back: later in the same sitting.
const againAfter = 10 * time.Minute

// NextReview is the stage and the day an entry is due after a review, and
// whether the learner has now learned it for good. A forgotten entry goes back
// to the start; a remembered one climbs a stage, an easy one two.
func NextReview(stage int, grade string, now time.Time) (next int, due time.Time, known bool) {
	switch grade {
	case GradeAgain:
		return 0, now.Add(againAfter), false
	case GradeEasy:
		stage += 2
	default:
		stage++
	}

	if stage < 0 {
		stage = 0
	}

	if stage > len(reviewDays) {
		return len(reviewDays), now.AddDate(0, 0, reviewDays[len(reviewDays)-1]), true
	}

	return stage, now.AddDate(0, 0, reviewDays[stage-1]), false
}

// ValidGrade reports whether grade is one a review accepts.
func ValidGrade(grade string) bool {
	return grade == GradeAgain || grade == GradeGood || grade == GradeEasy
}

// ReviewEntry records how one review of an entry went.
type ReviewEntry struct {
	UserID  uuid.UUID
	EntryID uuid.UUID
	Grade   string
}

type ReviewHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewReviewHandler(q *db.Queries, clk clock.Clock) ReviewHandler {
	return ReviewHandler{q: q, clock: clk}
}

func (h ReviewHandler) Handle(ctx context.Context, cmd ReviewEntry) (db.LearningEntry, error) {
	if !ValidGrade(cmd.Grade) {
		return db.LearningEntry{}, apperr.NewInvalid("grade must be again, good or easy")
	}

	row, err := h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(cmd.EntryID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	if err != nil {
		return row, err
	}

	now := h.clock.Now()
	stage, due, known := NextReview(int(row.ReviewStage), cmd.Grade, now)

	status := row.Status
	switch {
	case known:
		status = db.LearningEntryStatusKnown
	case status == db.LearningEntryStatusInbox:
		// Reviewing an entry is learning it.
		status = db.LearningEntryStatusLearning
	}

	return h.q.ReviewLearningEntry(ctx, db.ReviewLearningEntryParams{
		ID: row.ID, UserID: row.UserID, ReviewStage: int16(stage),
		NextReviewAt: db.Time(due), LastReviewedAt: db.Time(now), Status: status,
	})
}
