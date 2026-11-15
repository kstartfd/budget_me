package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// Daily sets: the learner picks how many new cards a day (DailyNew). Each day
// the next ones from the topic are put into that day's set; a day not opened is
// not lost, it adds to the credit, so the next set is bigger. Every calendar
// week and month the cards introduced in it can be reviewed again.
const (
	FrontTranslation = "translation" // the native side is shown first
	FrontText        = "text"        // the learned language is shown first

	DefaultDailyNew = 10
	MaxDailyNew     = 50
	// maxCredit keeps a long absence from becoming one endless set.
	maxCredit = 100
	// PeriodBatch is how many cards of a weekly or monthly review come at once;
	// the rest wait for the next sitting, and what was done stays done.
	PeriodBatch = 50

	PeriodDay   = "day"
	PeriodWeek  = "week"
	PeriodMonth = "month"
)

// LocalDate is the calendar day `now` is in the given time zone, as a date
// (UTC midnight), which is how dates are stored and compared.
func LocalDate(now time.Time, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" || timezone == "Local" {
		return time.Time{}, apperr.NewInvalid("tz must be an IANA zone such as Europe/Minsk")
	}

	local := now.In(loc)

	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC), nil
}

// WeekBounds is the Monday-to-Sunday week `day` is in.
func WeekBounds(day time.Time) (start, end time.Time) {
	offset := (int(day.Weekday()) + 6) % 7 // Monday is 0
	start = day.AddDate(0, 0, -offset)

	return start, start.AddDate(0, 0, 6)
}

// MonthBounds is the calendar month `day` is in.
func MonthBounds(day time.Time) (start, end time.Time) {
	start = time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)

	return start, start.AddDate(0, 1, -1)
}

// Accrue adds the quota of every day since `since` up to and including `today`
// to the credit, capped. `since` is the day the credit last accrued for.
func Accrue(credit, dailyNew int, since, today time.Time) int {
	days := int(today.Sub(since).Hours() / 24)
	if days <= 0 {
		return credit
	}

	return min(credit+days*dailyNew, maxCredit)
}

// Plan is a topic's daily-set settings.
type Plan struct {
	DailyNew int
	Front    string
}

// SetState is what a topic's sets look like today.
type SetState struct {
	Plan Plan
	// Available is how many entries wait to be introduced.
	Available int64
	Today     []db.LearningEntry
	Fresh     int64
	Weak      int64
	Week      Period
	Month     Period
}

// Period is a weekly or monthly review: its dates, how many cards belong to it
// and how many are still to go through.
type Period struct {
	From, To       time.Time
	Total, Pending int64
}

type SetsHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
}

func NewSetsHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock) SetsHandler {
	return SetsHandler{pool: pool, q: q, clock: clk}
}

func (h SetsHandler) topic(ctx context.Context, userID, topicID uuid.UUID) error {
	if _, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)}); err != nil {
		if db.IsNotFound(err) {
			return apperr.NewNotFound("topic not found")
		}

		return err
	}

	return nil
}

// Open brings the topic's plan up to today: it accrues the days that passed, puts
// as many entries into today's set as the credit allows, and returns the state.
// The plan row is locked meanwhile, so two tabs opening at once cannot hand out
// the day's cards twice.
func (h SetsHandler) Open(ctx context.Context, userID, topicID uuid.UUID, timezone string) (SetState, error) {
	today, err := LocalDate(h.clock.Now(), timezone)
	if err != nil {
		return SetState{}, err
	}

	if err := h.topic(ctx, userID, topicID); err != nil {
		return SetState{}, err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return SetState{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	q := h.q.WithTx(tx)
	now := db.Time(h.clock.Now())

	if err := q.EnsureLearningTopicPlan(ctx, db.EnsureLearningTopicPlanParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID), DailyNew: DefaultDailyNew, CreditedOn: db.Date(today), CreatedAt: now}); err != nil {
		return SetState{}, err
	}

	plan, err := q.GetLearningTopicPlanForUpdate(ctx, db.GetLearningTopicPlanForUpdateParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if err != nil {
		return SetState{}, err
	}

	credit := Accrue(int(plan.Credit), int(plan.DailyNew), plan.CreditedOn.Time, today)
	needTranslation := plan.Front == FrontTranslation

	if credit > 0 {
		took, err := q.IntroduceLearningEntries(ctx, db.IntroduceLearningEntriesParams{UserID: ids.ToPg(userID), TopicID: ids.ToPg(topicID), Day: db.Date(today), NeedTranslation: needTranslation, Take: int32(credit)})
		if err != nil {
			return SetState{}, err
		}

		credit -= int(took)
	}

	if err := q.SetLearningTopicPlanCredit(ctx, db.SetLearningTopicPlanCreditParams{TopicID: ids.ToPg(topicID), Credit: int32(credit), CreditedOn: db.Date(today), UpdatedAt: now}); err != nil {
		return SetState{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SetState{}, err
	}

	return h.state(ctx, userID, topicID, today, Plan{DailyNew: int(plan.DailyNew), Front: plan.Front})
}

func (h SetsHandler) state(ctx context.Context, userID, topicID uuid.UUID, today time.Time, plan Plan) (SetState, error) {
	uid, tid := ids.ToPg(userID), ids.ToPg(topicID)
	out := SetState{Plan: plan}

	var err error

	if out.Available, err = h.q.CountIntroducibleLearningEntries(ctx, db.CountIntroducibleLearningEntriesParams{UserID: uid, TopicID: tid, NeedTranslation: plan.Front == FrontTranslation}); err != nil {
		return out, err
	}

	if out.Today, err = h.q.ListLearningSetToday(ctx, db.ListLearningSetTodayParams{UserID: uid, TopicID: tid, Day: db.Date(today)}); err != nil {
		return out, err
	}

	counts, err := h.q.CountLearningSetToday(ctx, db.CountLearningSetTodayParams{UserID: uid, TopicID: tid, Day: db.Date(today)})
	if err != nil {
		return out, err
	}

	out.Fresh, out.Weak = counts.Fresh, counts.Weak

	ws, we := WeekBounds(today)

	week, err := h.q.CountLearningWeek(ctx, db.CountLearningWeekParams{UserID: uid, TopicID: tid, FromDate: db.Date(ws), ToDate: db.Date(we)})
	if err != nil {
		return out, err
	}

	out.Week = Period{From: ws, To: we, Total: week.Total, Pending: week.Pending}

	ms, me := MonthBounds(today)

	month, err := h.q.CountLearningMonth(ctx, db.CountLearningMonthParams{UserID: uid, TopicID: tid, FromDate: db.Date(ms), ToDate: db.Date(me)})
	if err != nil {
		return out, err
	}

	out.Month = Period{From: ms, To: me, Total: month.Total, Pending: month.Pending}

	return out, nil
}

// Pending returns the next batch of a weekly or monthly review.
func (h SetsHandler) Pending(ctx context.Context, userID, topicID uuid.UUID, period, timezone string) ([]db.LearningEntry, error) {
	today, err := LocalDate(h.clock.Now(), timezone)
	if err != nil {
		return nil, err
	}

	if err := h.topic(ctx, userID, topicID); err != nil {
		return nil, err
	}

	uid, tid := ids.ToPg(userID), ids.ToPg(topicID)

	switch period {
	case PeriodWeek:
		from, to := WeekBounds(today)

		return h.q.ListLearningWeekPending(ctx, db.ListLearningWeekPendingParams{UserID: uid, TopicID: tid, FromDate: db.Date(from), ToDate: db.Date(to), Take: PeriodBatch})
	case PeriodMonth:
		from, to := MonthBounds(today)

		return h.q.ListLearningMonthPending(ctx, db.ListLearningMonthPendingParams{UserID: uid, TopicID: tid, FromDate: db.Date(from), ToDate: db.Date(to), Take: PeriodBatch})
	default:
		return nil, apperr.NewInvalid("period must be week or month")
	}
}

// SetPlan changes how many cards a day and which side is shown first.
func (h SetsHandler) SetPlan(ctx context.Context, userID, topicID uuid.UUID, plan Plan) (Plan, error) {
	if plan.DailyNew < 1 || plan.DailyNew > MaxDailyNew || (plan.Front != FrontTranslation && plan.Front != FrontText) {
		return Plan{}, apperr.NewInvalid("dailyNew must be 1 to 50 and front translation or text")
	}

	if err := h.topic(ctx, userID, topicID); err != nil {
		return Plan{}, err
	}

	now := db.Time(h.clock.Now())

	// A topic that never opened its sets has no plan yet; make one first. Its
	// accrual date is corrected by the first Open, which carries the learner's zone.
	if err := h.q.EnsureLearningTopicPlan(ctx, db.EnsureLearningTopicPlanParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID), DailyNew: int16(plan.DailyNew), CreditedOn: db.Date(h.clock.Now().UTC().Truncate(24 * time.Hour)), CreatedAt: now}); err != nil {
		return Plan{}, err
	}

	row, err := h.q.SetLearningTopicPlanSettings(ctx, db.SetLearningTopicPlanSettingsParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID), DailyNew: int16(plan.DailyNew), Front: plan.Front, UpdatedAt: now})
	if err != nil {
		return Plan{}, err
	}

	return Plan{DailyNew: int(row.DailyNew), Front: row.Front}, nil
}

// GradeSet records how one card went in a set: known or not. In a weekly or
// monthly review it also marks the card as gone through for that period.
type GradeSet struct {
	UserID   uuid.UUID
	EntryID  uuid.UUID
	Period   string
	Known    bool
	Timezone string
}

func (h SetsHandler) Grade(ctx context.Context, cmd GradeSet) (db.LearningEntry, error) {
	today, err := LocalDate(h.clock.Now(), cmd.Timezone)
	if err != nil {
		return db.LearningEntry{}, err
	}

	if cmd.Period != PeriodDay && cmd.Period != PeriodWeek && cmd.Period != PeriodMonth {
		return db.LearningEntry{}, apperr.NewInvalid("period must be day, week or month")
	}

	row, err := h.q.GradeLearningSetEntry(ctx, db.GradeLearningSetEntryParams{ID: ids.ToPg(cmd.EntryID), UserID: ids.ToPg(cmd.UserID), Weak: !cmd.Known, Period: cmd.Period, Today: db.Date(today)})
	if db.IsNotFound(err) {
		return row, apperr.NewNotFound("entry not found")
	}

	return row, err
}
