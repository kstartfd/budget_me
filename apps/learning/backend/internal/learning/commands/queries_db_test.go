package commands_test

import (
	"context"
	"testing"
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/remind"
	"learning_app/backend/internal/testdb"
)

// The queries added for review, packs and reminders are plain SQL that sqlc only
// type-checks against the schema, not against Postgres' parameter typing. This runs
// each of them once so a mistake like an inconsistent parameter type fails here.
func TestReviewPackAndReminderQueriesRunAgainstARealDatabase(t *testing.T) {
	ctx := context.Background()
	pool, q := testdb.Pool(t)
	now := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	user := testdb.NewUser(t, pool, q, now, 1)
	topic := seed(t, ctx, q, user, now, 3)

	// Pack: what the topic holds, newest first.
	texts, err := q.ListLearningEntryTexts(ctx, db.ListLearningEntryTextsParams{UserID: ids.ToPg(user), TopicID: ids.ToPg(topic)})
	if err != nil || len(texts) != 3 {
		t.Fatalf("entry texts: %d %v", len(texts), err)
	}

	// Review: never-reviewed entries are due; grading one moves it on.
	reviews := commands.NewReviewHandler(q, testdb.Clock{At: now})

	due, err := q.CountDueLearningEntries(ctx, db.CountDueLearningEntriesParams{UserID: ids.ToPg(user), TopicID: ids.ToPg(topic), NextReviewAt: db.Time(now)})
	if err != nil || due.Fresh != 3 {
		t.Fatalf("due: %+v %v", due, err)
	}

	list, err := q.ListDueLearningEntries(ctx, db.ListDueLearningEntriesParams{UserID: ids.ToPg(user), TopicID: ids.ToPg(topic), NextReviewAt: db.Time(now), Limit: 20})
	if err != nil || len(list) != 3 {
		t.Fatalf("due list: %d %v", len(list), err)
	}

	row, err := reviews.Handle(ctx, commands.ReviewEntry{UserID: user, EntryID: ids.FromPg(list[0].ID), Grade: commands.GradeGood})
	if err != nil || row.ReviewStage != 1 || !row.NextReviewAt.Time.After(now) || row.Status != db.LearningEntryStatusLearning {
		t.Fatalf("reviewed: stage %d next %v status %s err %v", row.ReviewStage, row.NextReviewAt.Time, row.Status, err)
	}

	// Saving the same text again must not blank what is already filled in.
	entries := commands.NewEntryHandler(q, testdb.Clock{At: now})
	if _, err := entries.Handle(ctx, commands.SaveEntry{UserID: user, Fields: commands.EntryFields{TopicID: topic, Kind: db.LearningEntryKindWord, Text: list[0].Text, Status: db.LearningEntryStatusInbox}}); err != nil {
		t.Fatalf("save again: %v", err)
	}

	if kept, _ := q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: list[0].ID, UserID: ids.ToPg(user)}); kept.Translation != "слово" {
		t.Fatalf("a second save blanked the translation: %q", kept.Translation)
	}

	// Reminder: switch on, find it, count what is due, mark it sent.
	reminders := commands.NewReminderHandler(q, testdb.Clock{At: now})
	if _, err := reminders.Set(ctx, commands.SetReminder{UserID: user, Enabled: true, Timezone: "Europe/Minsk"}); err != nil {
		t.Fatalf("set reminder: %v", err)
	}

	if got, err := reminders.Get(ctx, user); err != nil || got == nil || !got.Enabled {
		t.Fatalf("get reminder: %+v %v", got, err)
	}

	count, err := q.CountLearningDueForUser(ctx, db.CountLearningDueForUserParams{UserID: ids.ToPg(user), NextReviewAt: db.Time(now)})
	if err != nil || count.Overdue+count.Fresh != 2 { // three entries, one reviewed into the future
		t.Fatalf("due for user: %+v %v", count, err)
	}

	if _, err := q.ListEnabledLearningReminders(ctx); err != nil {
		t.Fatalf("list enabled reminders: %v", err)
	}

	if err := q.MarkLearningReminderSent(ctx, db.MarkLearningReminderSentParams{UserID: ids.ToPg(user), LastSentOn: db.Date(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)), UpdatedAt: db.Time(now)}); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	// Once sent today, the same local day is not due again.
	reminder, _ := reminders.Get(ctx, user)
	if _, ok := remind.Due(now, "Europe/Minsk", &reminder.LastSentOn.Time); ok {
		t.Fatal("already sent today, must not be due again")
	}

	if err := q.DisableLearningReminder(ctx, db.DisableLearningReminderParams{UserID: ids.ToPg(user), UpdatedAt: db.Time(now)}); err != nil {
		t.Fatalf("disable: %v", err)
	}
}
