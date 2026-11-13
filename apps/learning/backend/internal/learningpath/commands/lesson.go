package commands

import (
	"context"
	"time"
	// The learner's browser names its zone (Europe/Moscow); a bare image has no
	// zone files to look that name up in.
	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// LessonLength is how long one sitting lasts. The clock starts with the lesson
// and does not wait for the learner: a pause only records that they stepped
// away.
const LessonLength = 30 * time.Minute

// LessonHandler runs the timed lessons of a learner. A lesson is the learner's
// own sitting, not a topic's: they move between the topics of the program while
// it runs, and what they answer in the meantime counts to it. A learner has one
// lesson at a time; one that ran past its end is closed at that end the next
// time anyone looks, so no timer has to run on the server.
type LessonHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewLessonHandler(q *db.Queries, clk clock.Clock) LessonHandler {
	return LessonHandler{q: q, clock: clk}
}

// Start opens a lesson, or hands back the one already running.
func (h LessonHandler) Start(ctx context.Context, userID uuid.UUID) (db.LearningPathLesson, error) {
	running, err := h.Active(ctx, userID)
	if err != nil {
		return db.LearningPathLesson{}, err
	}

	if running != nil {
		return *running, nil
	}

	now := h.clock.Now()

	return h.q.InsertLearningPathLesson(ctx, db.InsertLearningPathLessonParams{
		ID:        ids.ToPg(ids.New()),
		UserID:    ids.ToPg(userID),
		StartedAt: db.Time(now),
		EndsAt:    db.Time(now.Add(LessonLength)),
	})
}

// Active is the learner's running lesson, nil when there is none. A lesson
// whose time is up is finished at its end and counts as none.
func (h LessonHandler) Active(ctx context.Context, userID uuid.UUID) (*db.LearningPathLesson, error) {
	lesson, err := h.q.GetActiveLearningPathLesson(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if h.clock.Now().Before(lesson.EndsAt.Time) {
		return &lesson, nil
	}

	if _, err := h.q.FinishLearningPathLesson(ctx, db.FinishLearningPathLessonParams{ID: lesson.ID, UserID: lesson.UserID, FinishedAt: lesson.EndsAt}); err != nil && !db.IsNotFound(err) {
		return nil, err
	}

	return nil, nil
}

// Pause marks the learner as away; the lesson keeps running.
func (h LessonHandler) Pause(ctx context.Context, userID, lessonID uuid.UUID) (db.LearningPathLesson, error) {
	if err := h.settle(ctx, userID, lessonID); err != nil {
		return db.LearningPathLesson{}, err
	}

	lesson, err := h.q.PauseLearningPathLesson(ctx, db.PauseLearningPathLessonParams{ID: ids.ToPg(lessonID), UserID: ids.ToPg(userID), PausedAt: db.Time(h.clock.Now())})
	if db.IsNotFound(err) {
		return h.current(ctx, userID, lessonID)
	}

	return lesson, err
}

// Resume brings the learner back to a paused lesson.
func (h LessonHandler) Resume(ctx context.Context, userID, lessonID uuid.UUID) (db.LearningPathLesson, error) {
	if err := h.settle(ctx, userID, lessonID); err != nil {
		return db.LearningPathLesson{}, err
	}

	lesson, err := h.q.ResumeLearningPathLesson(ctx, db.ResumeLearningPathLessonParams{ID: ids.ToPg(lessonID), UserID: ids.ToPg(userID), Column3: db.Time(h.clock.Now())})
	if db.IsNotFound(err) {
		return h.current(ctx, userID, lessonID)
	}

	return lesson, err
}

// Finish ends the lesson now. Finishing a finished lesson changes nothing.
func (h LessonHandler) Finish(ctx context.Context, userID, lessonID uuid.UUID) (db.LearningPathLesson, error) {
	if err := h.settle(ctx, userID, lessonID); err != nil {
		return db.LearningPathLesson{}, err
	}

	lesson, err := h.q.FinishLearningPathLesson(ctx, db.FinishLearningPathLessonParams{ID: ids.ToPg(lessonID), UserID: ids.ToPg(userID), FinishedAt: db.Time(h.clock.Now())})
	if db.IsNotFound(err) {
		return h.current(ctx, userID, lessonID)
	}

	return lesson, err
}

// settle closes the lesson if its time is up, so a late pause, resume or
// finish sees it finished at its real end.
func (h LessonHandler) settle(ctx context.Context, userID, lessonID uuid.UUID) error {
	lesson, err := h.current(ctx, userID, lessonID)
	if err != nil {
		return err
	}

	if lesson.Status == db.LearningPathLessonStatusActive && !h.clock.Now().Before(lesson.EndsAt.Time) {
		_, err = h.q.FinishLearningPathLesson(ctx, db.FinishLearningPathLessonParams{ID: lesson.ID, UserID: lesson.UserID, FinishedAt: lesson.EndsAt})
		if db.IsNotFound(err) {
			return nil
		}
	}

	return err
}

func (h LessonHandler) current(ctx context.Context, userID, lessonID uuid.UUID) (db.LearningPathLesson, error) {
	lesson, err := h.q.GetLearningPathLesson(ctx, db.GetLearningPathLessonParams{ID: ids.ToPg(lessonID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return db.LearningPathLesson{}, apperr.NewNotFound("lesson not found")
	}

	return lesson, err
}

// Stats is what a lesson came to: the attempts made in it and how many were right.
type Stats struct {
	Attempts int
	Correct  int
}

// List is the learner's lessons, newest first, each with its stats.
func (h LessonHandler) List(ctx context.Context, userID uuid.UUID) ([]db.LearningPathLesson, map[uuid.UUID]Stats, error) {
	// Settle an expired lesson first so the list shows it finished.
	if _, err := h.Active(ctx, userID); err != nil {
		return nil, nil, err
	}

	lessons, err := h.q.ListLearningPathLessons(ctx, ids.ToPg(userID))
	if err != nil {
		return nil, nil, err
	}

	stats, err := h.Stats(ctx, lessons)

	return lessons, stats, err
}

// Stats counts the attempts of lessons, keyed by lesson id.
func (h LessonHandler) Stats(ctx context.Context, lessons []db.LearningPathLesson) (map[uuid.UUID]Stats, error) {
	out := make(map[uuid.UUID]Stats, len(lessons))
	if len(lessons) == 0 {
		return out, nil
	}

	idList := make([]pgtype.UUID, len(lessons))
	for i, l := range lessons {
		idList[i] = l.ID
	}

	rows, err := h.q.ListLearningPathLessonStats(ctx, idList)
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		out[ids.FromPg(r.LessonID)] = Stats{Attempts: int(r.Attempts), Correct: int(r.Correct)}
	}

	return out, nil
}

// Totals is the time a learner has studied in lessons today, this week (from
// Monday) and this month. A day is the learner's own, so it needs their zone.
type Totals struct {
	Today time.Duration
	Week  time.Duration
	Month time.Duration
}

// LocationOf is the zone a browser reported, UTC when it is empty or unknown:
// a wrong zone only moves the midnight, it must not fail the request.
func LocationOf(name string) *time.Location {
	if name == "" || len(name) > 64 {
		return time.UTC
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}

// Totals adds up the learner's lesson time. A lesson counts for the day, week
// and month its start falls in, and for the time spent at the desk, not for the
// pauses.
func (h LessonHandler) Totals(ctx context.Context, userID uuid.UUID, loc *time.Location) (Totals, error) {
	now := h.clock.Now()
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	sinceMonday := (int(local.Weekday()) + 6) % 7
	week := day.AddDate(0, 0, -sinceMonday)
	month := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)

	row, err := h.q.SumLearningLessonSeconds(ctx, db.SumLearningLessonSecondsParams{
		UserID:     ids.ToPg(userID),
		Now:        db.Time(now),
		DayStart:   db.Time(day),
		WeekStart:  db.Time(week),
		MonthStart: db.Time(month),
	})
	if err != nil {
		return Totals{}, err
	}

	return Totals{
		Today: time.Duration(row.Today) * time.Second,
		Week:  time.Duration(row.Week) * time.Second,
		Month: time.Duration(row.Month) * time.Second,
	}, nil
}
