package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningprofile"
)

// maxOpenReviews is how many reviews are offered at once: a backlog is worked
// off a few at a time, not shown as a wall.
const maxOpenReviews = 5

// PlannedActivity is an open learning activity with what a screen needs to show it.
type PlannedActivity struct {
	Activity db.LearningActivity
	Item     *db.LearningKnowledgeItem
	// Title and Goal are in the language the teacher explains in; empty when the
	// activity is not tied to a library topic.
	Title string
	Goal  string
	Topic *db.LearningPathTopic
	// Exercise is the exercise a review asks; nil for any other activity.
	Exercise *db.LearningPathExercise
	// State is the item's knowledge state as the API names it, Lapsed when it
	// was learnt and then forgotten.
	State  string
	Lapsed bool
	DueAt  *time.Time
}

// ActivityHandler plans what the learner should do now: the reviews that are due
// and the topic of the running program they are on. Planning is safe to repeat.
type ActivityHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewActivityHandler(q *db.Queries, clk clock.Clock) ActivityHandler {
	return ActivityHandler{q: q, clock: clk}
}

// Plan brings the learner's activities up to date and returns the open ones,
// reviews first. It closes what has been done or has gone stale since the last
// look, and opens a review for each due item that has an exercise to ask and a
// practice for the topic being worked on.
func (h ActivityHandler) Plan(ctx context.Context, userID uuid.UUID, language string) ([]PlannedActivity, db.LearningTeacher, error) {
	user := ids.ToPg(userID)

	teacher, err := h.q.GetLearningTeacherByLanguage(ctx, db.GetLearningTeacherByLanguageParams{UserID: user, Language: language})
	if db.IsNotFound(err) {
		return nil, db.LearningTeacher{}, apperr.NewNotFound("teacher not found")
	}

	if err != nil {
		return nil, db.LearningTeacher{}, err
	}

	now := h.clock.Now()
	at := db.Time(now)

	if err := h.settle(ctx, teacher, at); err != nil {
		return nil, teacher, err
	}

	if err := h.openReviews(ctx, teacher, language, now); err != nil {
		return nil, teacher, err
	}

	if err := h.openPractice(ctx, teacher, language, now); err != nil {
		return nil, teacher, err
	}

	open, err := h.q.ListOpenLearningActivities(ctx, db.ListOpenLearningActivitiesParams{TeacherID: teacher.ID, UserID: user})
	if err != nil {
		return nil, teacher, err
	}

	out := make([]PlannedActivity, 0, len(open))

	for _, a := range open {
		planned, ok, err := h.describe(ctx, teacher, language, a)
		if err != nil {
			return nil, teacher, err
		}

		if ok {
			out = append(out, planned)
		}
	}

	return out, teacher, nil
}

// settle closes the activities that no longer stand: a practice whose topic is
// finished (done) or whose program was closed (skipped), a review of an item
// that is not due any more (skipped).
func (h ActivityHandler) settle(ctx context.Context, teacher db.LearningTeacher, at pgtype.Timestamptz) error {
	if _, err := h.q.CloseFinishedPracticeActivities(ctx, db.CloseFinishedPracticeActivitiesParams{TeacherID: teacher.ID, UserID: teacher.UserID, CompletedAt: at}); err != nil {
		return err
	}

	if _, err := h.q.SkipStalePracticeActivities(ctx, db.SkipStalePracticeActivitiesParams{TeacherID: teacher.ID, UserID: teacher.UserID, CompletedAt: at}); err != nil {
		return err
	}

	_, err := h.q.SkipStaleReviewActivities(ctx, db.SkipStaleReviewActivitiesParams{TeacherID: teacher.ID, UserID: teacher.UserID, CompletedAt: at})

	return err
}

func (h ActivityHandler) openReviews(ctx context.Context, teacher db.LearningTeacher, language string, now time.Time) error {
	open, err := h.q.ListOpenLearningActivities(ctx, db.ListOpenLearningActivitiesParams{TeacherID: teacher.ID, UserID: teacher.UserID})
	if err != nil {
		return err
	}

	// What is already offered counts against the cap, so a backlog is worked off
	// a few at a time however often the plan is read.
	offered := map[uuid.UUID]bool{}

	for _, a := range open {
		if a.Type == "review" {
			offered[uuidOf(a.ItemID)] = true
		}
	}

	due, err := h.q.ListDueReviewItems(ctx, db.ListDueReviewItemsParams{TeacherID: teacher.ID, UserID: teacher.UserID, DueAt: db.Time(now), Limit: maxOpenReviews})
	if err != nil {
		return err
	}

	for _, d := range due {
		if offered[uuidOf(d.ItemID)] {
			continue
		}

		if len(offered) >= maxOpenReviews {
			break
		}

		goal := ""
		if lib, ok := curriculum.Find(language, ExplainLanguage(teacher), d.Key); ok {
			goal = lib.Goal
		}

		opened, err := h.q.InsertLearningActivity(ctx, db.InsertLearningActivityParams{
			ID: ids.ToPg(ids.New()), UserID: teacher.UserID, TeacherID: teacher.ID,
			Type: "review", Reason: "due", Goal: goal, ItemID: d.ItemID, Params: []byte(`{}`), CreatedAt: db.Time(now),
		})
		if err != nil {
			return err
		}

		if opened > 0 {
			offered[uuidOf(d.ItemID)] = true
		}
	}

	return nil
}

func (h ActivityHandler) openPractice(ctx context.Context, teacher db.LearningTeacher, language string, now time.Time) error {
	topic, err := h.q.GetCurrentPracticeTopic(ctx, db.GetCurrentPracticeTopicParams{UserID: teacher.UserID, Language: language})
	if db.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	var item db.LearningKnowledgeItem

	if topic.SourceID != "" {
		made, _, ok, err := ensureItem(ctx, h.q, teacher, language, topic.SourceID, now)
		if err != nil {
			return err
		}

		if ok {
			item = made
		}
	}

	_, err = h.q.InsertLearningActivity(ctx, db.InsertLearningActivityParams{
		ID: ids.ToPg(ids.New()), UserID: teacher.UserID, TeacherID: teacher.ID,
		Type: "practice", Reason: "path", Goal: topic.Goal, ItemID: item.ID, TopicID: topic.ID, Params: []byte(`{}`), CreatedAt: db.Time(now),
	})

	return err
}

// describe adds to an open activity what a screen shows. ok is false for a
// review whose exercise has gone (the history was cleared): it is not offered.
func (h ActivityHandler) describe(ctx context.Context, teacher db.LearningTeacher, language string, a db.LearningActivity) (PlannedActivity, bool, error) {
	out := PlannedActivity{Activity: a, Goal: a.Goal}

	if a.ItemID.Valid {
		item, err := h.q.GetKBItem(ctx, a.ItemID)
		if err != nil {
			return out, false, err
		}

		out.Item = &item

		if lib, ok := curriculum.Find(language, ExplainLanguage(teacher), item.Key); ok {
			out.Title = lib.Title
		}

		state, err := h.q.GetLearnerItemState(ctx, db.GetLearnerItemStateParams{TeacherID: a.TeacherID, ItemID: a.ItemID, UserID: a.UserID})
		if err != nil && !db.IsNotFound(err) {
			return out, false, err
		}

		if err == nil {
			out.State, out.Lapsed = learningprofile.Public(learningprofile.State(state.State))
			out.DueAt = db.TimePtr(state.DueAt)
		}
	}

	if a.TopicID.Valid {
		topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: a.TopicID, UserID: a.UserID})
		if err != nil {
			return out, false, err
		}

		out.Topic = &topic

		if out.Title == "" {
			out.Title = topic.Title
		}
	}

	if a.Type == "review" {
		if out.Item == nil {
			return out, false, nil
		}

		exercise, err := h.q.PickReviewExercise(ctx, db.PickReviewExerciseParams{UserID: a.UserID, Language: language, SourceID: out.Item.Key})
		if db.IsNotFound(err) {
			return out, false, nil
		}

		if err != nil {
			return out, false, err
		}

		out.Exercise = &exercise
	}

	return out, true, nil
}
