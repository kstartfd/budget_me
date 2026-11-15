// Package remind sends the daily "cards are waiting" message through the
// Telegram bot to the learners who switched it on. It runs once an hour from
// cron (cmd/remind); a learner gets at most one message a day, in their own
// morning, and only when something is actually due.
package remind

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/tgbot"
)

const (
	// firstHour and lastHour bound the local hours a reminder may go out. The job
	// runs hourly, so a missed run still catches up later the same day.
	firstHour = 9
	lastHour  = 20
	// freshCap: never-reviewed cards count only up to this many, so a long inbox
	// does not read as a mountain.
	freshCap = 10
)

// Sender is the bot.
type Sender interface {
	Notify(ctx context.Context, chatID int64, text, button string) error
}

// Due reports whether a reminder may go out to a learner now, and the local date
// to record. It is when their local clock is inside the sending hours and no
// reminder has gone out on that local date.
func Due(now time.Time, timezone string, lastSent *time.Time) (time.Time, bool) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, false
	}

	local := now.In(loc)
	if local.Hour() < firstHour || local.Hour() > lastHour {
		return time.Time{}, false
	}

	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	if lastSent != nil && lastSent.UTC().Equal(date) {
		return date, false
	}

	return date, true
}

// Message is the text and the button label for n cards, in the learner's
// language (ru, es, or English for anything else).
func Message(lang string, n int) (text, button string) {
	switch strings.ToLower(strings.SplitN(lang, "-", 2)[0]) {
	case "ru":
		return fmt.Sprintf("Пора повторить слова: %s ждут вас.", plural(n, "карточка", "карточки", "карточек")), "📚 Повторить"
	case "es":
		if n == 1 {
			return "Es hora de repasar: te espera 1 tarjeta.", "📚 Repasar"
		}

		return fmt.Sprintf("Es hora de repasar: te esperan %d tarjetas.", n), "📚 Repasar"
	default:
		if n == 1 {
			return "Time to review: 1 card is waiting for you.", "📚 Review"
		}

		return fmt.Sprintf("Time to review: %d cards are waiting for you.", n), "📚 Review"
	}
}

// plural spells n with the Russian form: 1 карточка, 2 карточки, 5 карточек.
func plural(n int, one, few, many string) string {
	form := many

	switch m10, m100 := n%10, n%100; {
	case m10 == 1 && m100 != 11:
		form = one
	case m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14):
		form = few
	}

	return fmt.Sprintf("%d %s", n, form)
}

// Run sends every reminder that is due at `now` and returns how many went out.
// One learner's failure never stops the others.
func Run(ctx context.Context, q *db.Queries, send Sender, now time.Time) (int, error) {
	rows, err := q.ListEnabledLearningReminders(ctx)
	if err != nil {
		return 0, err
	}

	sent := 0

	for _, r := range rows {
		var last *time.Time
		if r.LastSentOn.Valid {
			t := r.LastSentOn.Time
			last = &t
		}

		date, ok := Due(now, r.Timezone, last)
		if !ok {
			continue
		}

		due, err := q.CountLearningDueForUser(ctx, db.CountLearningDueForUserParams{UserID: r.UserID, NextReviewAt: db.Time(now)})
		if err != nil {
			slog.Warn("remind: counting due cards", "user_id", ids.FromPg(r.UserID), "err", err)

			continue
		}

		n := int(due.Overdue) + min(int(due.Fresh), freshCap)
		if n == 0 {
			continue
		}

		text, button := Message(r.LanguageCode.String, n)

		switch err := send.Notify(ctx, r.TelegramID.Int64, text, button); {
		case errors.Is(err, tgbot.ErrBlocked):
			slog.Info("remind: the user cannot be messaged, switching the reminder off", "user_id", ids.FromPg(r.UserID))

			_ = q.DisableLearningReminder(ctx, db.DisableLearningReminderParams{UserID: r.UserID, UpdatedAt: db.Time(now)})
		case err != nil:
			slog.Warn("remind: sending", "user_id", ids.FromPg(r.UserID), "err", err)
		default:
			if err := q.MarkLearningReminderSent(ctx, db.MarkLearningReminderSentParams{UserID: r.UserID, LastSentOn: db.Date(date), UpdatedAt: db.Time(now)}); err != nil {
				slog.Warn("remind: recording the send", "user_id", ids.FromPg(r.UserID), "err", err)
			}

			sent++
		}
	}

	return sent, nil
}
