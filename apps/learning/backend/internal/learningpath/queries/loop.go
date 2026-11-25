package queries

import (
	"context"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningprofile"
)

// Dashboard is the minimal picture of the learning loop for one language: what
// the learner knows, what is due, and how much they have shown.
type Dashboard struct {
	// Known counts the items the learner has shown something about, by the
	// state the API names them; Lapsed counts those that were learnt and forgotten.
	Known   map[string]int
	Tracked int
	Lapsed  int
	Reviews ReviewCounts
	// Evidence counts what the learner has shown, in all and in the last week.
	EvidenceTotal  int
	EvidenceRecent int
}

// ReviewCounts is the review queue read from the due times.
type ReviewCounts struct {
	Due      int
	Overdue  int
	Upcoming int
	NextDue  *time.Time
}

// weekAgo is how far back "recently" reaches.
const weekAgo = 7 * 24 * time.Hour

// Dashboard reads the counts of one teacher's loop. Nothing in it is estimated:
// every number is a count of stored rows, and a learner with none sees zeros.
func (h ReadHandler) Dashboard(ctx context.Context, userID uuid.UUID, teacher db.LearningTeacher, now time.Time, rules learningprofile.Config) (Dashboard, error) {
	out := Dashboard{Known: map[string]int{"learning": 0, "practiced": 0, "mastered": 0}}
	user := ids.ToPg(userID)

	states, err := h.q.CountLearnerItemStates(ctx, db.CountLearnerItemStatesParams{TeacherID: teacher.ID, UserID: user})
	if err != nil {
		return out, err
	}

	for _, s := range states {
		name, lapsed := learningprofile.Public(learningprofile.State(s.State))
		out.Known[name] += int(s.Count)
		out.Tracked += int(s.Count)

		if lapsed {
			out.Lapsed += int(s.Count)
		}
	}

	reviews, err := h.q.CountLearnerReviews(ctx, db.CountLearnerReviewsParams{
		TeacherID: teacher.ID, UserID: user, Now: db.Time(now), OverdueBefore: db.Time(now.Add(-rules.OverdueAfter)),
	})
	if err != nil {
		return out, err
	}

	out.Reviews = ReviewCounts{Due: int(reviews.Due), Overdue: int(reviews.Overdue), Upcoming: int(reviews.Upcoming), NextDue: db.TimePtr(reviews.NextDue)}

	evidence, err := h.q.CountLearningEvidence(ctx, db.CountLearningEvidenceParams{TeacherID: teacher.ID, UserID: user, Since: db.Time(now.Add(-weekAgo))})
	if err != nil {
		return out, err
	}

	out.EvidenceTotal, out.EvidenceRecent = int(evidence.Total), int(evidence.Recent)

	return out, nil
}
