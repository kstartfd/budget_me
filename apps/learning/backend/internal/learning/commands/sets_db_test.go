package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/testdb"
)

// seed makes a topic with n entries that already have a translation.
func seed(t *testing.T, ctx context.Context, q *db.Queries, user uuid.UUID, now time.Time, n int) uuid.UUID {
	t.Helper()

	topic := ids.New()
	if _, err := q.InsertLearningTopic(ctx, db.InsertLearningTopicParams{ID: ids.ToPg(topic), UserID: ids.ToPg(user), Name: "Sets", Language: "en", TranslationLanguage: "ru", CreatedAt: db.Time(now)}); err != nil {
		t.Fatalf("topic: %v", err)
	}

	entries := commands.NewEntryHandler(q, testdb.Clock{At: now})

	for i := 0; i < n; i++ {
		_, err := entries.Handle(ctx, commands.SaveEntry{UserID: user, Fields: commands.EntryFields{
			TopicID: topic, Kind: db.LearningEntryKindWord, Text: "word" + string(rune('a'+i)), Translation: "слово", Status: db.LearningEntryStatusInbox,
		}})
		if err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
	}

	return topic
}

func TestDailySetsAgainstARealDatabase(t *testing.T) {
	ctx := context.Background()
	pool, q := testdb.Pool(t)
	now := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC) // a Monday
	user := testdb.NewUser(t, pool, q, now, 1)
	topic := seed(t, ctx, q, user, now, 25)

	day := func(d int) commands.SetsHandler {
		return commands.NewSetsHandler(pool, q, testdb.Clock{At: now.AddDate(0, 0, d)})
	}

	// Day one: the default ten, and opening again hands out no more.
	state, err := day(0).Open(ctx, user, topic, "UTC")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if len(state.Today) != 10 || state.Fresh != 10 || state.Available != 15 {
		t.Fatalf("day one: today %d fresh %d available %d", len(state.Today), state.Fresh, state.Available)
	}

	if again, _ := day(0).Open(ctx, user, topic, "UTC"); len(again.Today) != 10 {
		t.Fatalf("opening twice must not hand out more, got %d", len(again.Today))
	}

	// Two days skipped: three days' quota is owed, but only 15 words are left.
	state, err = day(3).Open(ctx, user, topic, "UTC")
	if err != nil {
		t.Fatalf("open after a gap: %v", err)
	}

	if state.Fresh != 15 || state.Available != 0 {
		t.Fatalf("after the gap: fresh %d available %d", state.Fresh, state.Available)
	}

	// Grading: one card not known comes back with the next day's set.
	card := state.Today[0]
	if _, err := day(3).Grade(ctx, commands.GradeSet{UserID: user, EntryID: ids.FromPg(card.ID), Period: commands.PeriodDay, Known: false, Timezone: "UTC"}); err != nil {
		t.Fatalf("grade: %v", err)
	}

	state, _ = day(4).Open(ctx, user, topic, "UTC")
	if state.Weak != 1 {
		t.Fatalf("the card not known must come back, weak=%d", state.Weak)
	}

	// The week of Mon 16 to Sun 22 holds all 25; reviewing one in the week marks it done.
	if state.Week.Total != 25 || state.Week.Pending != 25 {
		t.Fatalf("week: %+v", state.Week)
	}

	batch, err := day(4).Pending(ctx, user, topic, commands.PeriodWeek, "UTC")
	if err != nil || len(batch) != 25 {
		t.Fatalf("week batch: %d %v", len(batch), err)
	}

	if _, err := day(4).Grade(ctx, commands.GradeSet{UserID: user, EntryID: ids.FromPg(batch[0].ID), Period: commands.PeriodWeek, Known: true, Timezone: "UTC"}); err != nil {
		t.Fatalf("grade week: %v", err)
	}

	state, _ = day(4).Open(ctx, user, topic, "UTC")
	if state.Week.Pending != 24 || state.Month.Total != 25 {
		t.Fatalf("after one week review: week %+v month %+v", state.Week, state.Month)
	}

	// The plan can be changed.
	plan, err := day(4).SetPlan(ctx, user, topic, commands.Plan{DailyNew: 20, Front: commands.FrontText})
	if err != nil || plan.DailyNew != 20 || plan.Front != commands.FrontText {
		t.Fatalf("plan: %+v %v", plan, err)
	}
}
