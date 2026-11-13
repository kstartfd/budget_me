package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/testdb"
)

func moscow(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	return loc
}

// lessonRow writes a lesson as the server would have left it.
func lessonRow(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, status string, started, ends time.Time, finished, pausedAt *time.Time, pausedSeconds int) {
	t.Helper()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO learning_path_lessons (id, user_id, status, started_at, ends_at, finished_at, paused_at, paused_seconds)
		 VALUES ($1, $2, $3::learning_path_lesson_status, $4, $5, $6, $7, $8)`,
		ids.New(), user, status, started, ends, finished, pausedAt, pausedSeconds)
	if err != nil {
		t.Fatal(err)
	}
}

func at(t *testing.T, loc *time.Location, layout string) time.Time {
	t.Helper()

	moment, err := time.ParseInLocation("2006-01-02 15:04", layout, loc)
	if err != nil {
		t.Fatal(err)
	}

	return moment
}

// TestLessonTimeIsAddedUpPerDayWeekAndMonthOfTheLearner: now is Wednesday
// 11 November 2026, 14:00 in Moscow. A lesson counts for the windows its start
// falls in, in the learner's zone, at the time they spent at the desk.
func TestLessonTimeIsAddedUpPerDayWeekAndMonthOfTheLearner(t *testing.T) {
	pool, q := testdb.Pool(t)
	msk := moscow(t)
	now := at(t, msk, "2026-11-11 14:00")
	user := testdb.NewUser(t, pool, q, now, 1)
	handler := commands.NewLessonHandler(q, testdb.Clock{At: now})

	finished := func(start, end string, paused int) {
		s, e := at(t, msk, start), at(t, msk, end)
		lessonRow(t, pool, user, "finished", s, s.Add(commands.LessonLength), &e, nil, paused)
	}

	// Today: 25 minutes with 5 of them paused, and 10 minutes just after midnight.
	finished("2026-11-11 10:00", "2026-11-11 10:25", 300)
	finished("2026-11-11 00:30", "2026-11-11 00:40", 0)
	// Still running, paused for the last four minutes: 10 minutes in, 1 paused
	// before and 4 now, so 5 at the desk.
	started, pausedAt := at(t, msk, "2026-11-11 13:50"), at(t, msk, "2026-11-11 13:56")
	lessonRow(t, pool, user, "active", started, started.Add(commands.LessonLength), nil, &pausedAt, 60)
	// Monday of this week, Sunday before it, and the month before.
	finished("2026-11-09 09:00", "2026-11-09 09:30", 0)
	finished("2026-11-08 20:00", "2026-11-08 20:30", 0)
	finished("2026-10-31 23:00", "2026-10-31 23:30", 0)

	got, err := handler.Totals(context.Background(), user, msk)
	if err != nil {
		t.Fatal(err)
	}

	if want := (commands.Totals{Today: 35 * time.Minute, Week: 65 * time.Minute, Month: 95 * time.Minute}); got != want {
		t.Fatalf("in Moscow: %+v, want %+v", got, want)
	}

	// The same lessons for a learner in UTC: the one after Moscow's midnight
	// started the evening before, so it is not today's, but it is this week's.
	got, err = handler.Totals(context.Background(), user, time.UTC)
	if err != nil {
		t.Fatal(err)
	}

	if want := (commands.Totals{Today: 25 * time.Minute, Week: 65 * time.Minute, Month: 95 * time.Minute}); got != want {
		t.Fatalf("in UTC: %+v, want %+v", got, want)
	}

	// Someone else's lessons are not counted.
	other := testdb.NewUser(t, pool, q, now, 1)
	if got, err := handler.Totals(context.Background(), other, msk); err != nil || got != (commands.Totals{}) {
		t.Fatalf("another learner: %+v %v", got, err)
	}
}

// TestALessonPastItsEndIsCountedToItsEnd: a lesson nobody closed stops at the
// end the server set for it, not at the moment someone looks.
func TestALessonPastItsEndIsCountedToItsEnd(t *testing.T) {
	pool, q := testdb.Pool(t)
	msk := moscow(t)
	now := at(t, msk, "2026-11-11 18:00")
	user := testdb.NewUser(t, pool, q, now, 1)

	started := at(t, msk, "2026-11-11 10:00")
	lessonRow(t, pool, user, "active", started, started.Add(commands.LessonLength), nil, nil, 0)

	got, err := commands.NewLessonHandler(q, testdb.Clock{At: now}).Totals(context.Background(), user, msk)
	if err != nil {
		t.Fatal(err)
	}

	if got.Today != commands.LessonLength {
		t.Fatalf("an unclosed lesson counts %v, want its length %v", got.Today, commands.LessonLength)
	}
}

func TestAnUnknownZoneFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Mars/Olympus", "../../etc/passwd", "Europe/Moscow"} {
		loc := commands.LocationOf(name)
		if name == "Europe/Moscow" {
			if loc.String() != name {
				t.Errorf("%q resolved to %v", name, loc)
			}

			continue
		}

		if loc != time.UTC {
			t.Errorf("%q resolved to %v, want UTC", name, loc)
		}
	}

	if loc := commands.LocationOf(string(make([]byte, 100))); loc != time.UTC {
		t.Errorf("an overlong name must be ignored, got %v", loc)
	}
}
