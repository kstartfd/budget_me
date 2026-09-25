package dayemail

import (
	"context"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/resend"
)

type SweepResult struct {
	Sent   int
	Failed int
}

// Sweep sends one frozen digest per user, local day and slot. A separate cron
// invokes it every 15 minutes; it is safe to retry after a worker crash.
func Sweep(ctx context.Context, q *db.Queries, now time.Time, mail *resend.Client, appURL string) (SweepResult, error) {
	var result SweepResult
	if !mail.Enabled() {
		return result, nil
	}
	subs, err := q.ListActiveDayEmailSubscriptions(ctx)
	if err != nil {
		return result, err
	}
	for _, sub := range subs {
		loc, err := time.LoadLocation(sub.Timezone)
		if err != nil {
			result.Failed++
			fmt.Printf("day-email: user=%s invalid timezone\n", ids.FromPg(sub.UserID))
			continue
		}
		local := now.In(loc)
		slot, due := dueSlot(local, sub.Frequency)
		if !due {
			continue
		}
		day, _ := db.ParseDate(local.Format("2006-01-02"))
		key := db.GetDayEmailDeliveryParams{UserID: sub.UserID, LocalDay: day, Slot: slot}
		delivery, err := q.GetDayEmailDelivery(ctx, key)
		if err != nil && !db.IsNotFound(err) {
			result.Failed++
			fmt.Printf("day-email: user=%s read delivery failed: %v\n", ids.FromPg(sub.UserID), err)
			continue
		}
		if db.IsNotFound(err) {
			tasks, err := q.ListOpenTasksForDigest(ctx, db.ListOpenTasksForDigestParams{
				UserID: sub.UserID, LocalDay: day,
			})
			if err != nil {
				result.Failed++
				fmt.Printf("day-email: user=%s read tasks failed: %v\n", ids.FromPg(sub.UserID), err)
				continue
			}
			if len(tasks) == 0 {
				continue
			}
			subject, body := digestText(tasks, db.DateStr(day), slot, db.TextPtr(sub.LanguageCode), strings.TrimRight(appURL, "/"))
			_, err = q.InsertDayEmailDelivery(ctx, db.InsertDayEmailDeliveryParams{
				UserID: sub.UserID, LocalDay: day, Slot: slot, Email: sub.Email,
				Subject: subject, Body: body, Now: db.Time(now),
			})
			if err == nil {
				delivery, err = q.GetDayEmailDelivery(ctx, key)
			}
			if err != nil {
				result.Failed++
				fmt.Printf("day-email: user=%s create delivery failed: %v\n", ids.FromPg(sub.UserID), err)
				continue
			}
		}
		if delivery.SentAt.Valid || delivery.Email != sub.Email {
			continue
		}
		// Respect a preference changed while this sweep was assembling the
		// digest. In particular, a disabled subscription must not be sent.
		current, err := q.GetDayEmailSubscription(ctx, sub.UserID)
		if db.IsNotFound(err) {
			continue
		}
		if err != nil {
			result.Failed++
			fmt.Printf("day-email: user=%s recheck failed: %v\n", ids.FromPg(sub.UserID), err)
			continue
		}
		if !current.ConfirmedAt.Valid || current.Email != sub.Email || (slot == db.DayEmailSlotEvening && current.Frequency != db.DayEmailFrequencyTwice) {
			continue
		}
		idempotencyKey := "day-digest-" + ids.FromPg(sub.UserID).String() + "-" + db.DateStr(day) + "-" + string(slot)
		messageID, err := mail.Send(ctx, delivery.Email, delivery.Subject, delivery.Body, idempotencyKey)
		if err != nil {
			result.Failed++
			fmt.Printf("day-email: user=%s send failed: %v\n", ids.FromPg(sub.UserID), err)
			continue
		}
		if err := q.MarkDayEmailDeliverySent(ctx, db.MarkDayEmailDeliverySentParams{
			UserID: sub.UserID, LocalDay: day, Slot: slot,
			ResendID: db.Text(&messageID), Now: db.Time(now),
		}); err != nil {
			result.Failed++
			fmt.Printf("day-email: user=%s mark sent failed: %v\n", ids.FromPg(sub.UserID), err)
			continue
		}
		result.Sent++
	}
	return result, nil
}

// The morning digest is sent between 09:00 and 12:00 local time. The optional
// second digest is sent between 18:00 and 21:00; outside those windows the
// worker leaves the user alone instead of delivering a stale late-night mail.
func dueSlot(local time.Time, frequency db.DayEmailFrequency) (db.DayEmailSlot, bool) {
	if local.Hour() >= 9 && local.Hour() < 12 {
		return db.DayEmailSlotMorning, true
	}
	if frequency == db.DayEmailFrequencyTwice && local.Hour() >= 18 && local.Hour() < 21 {
		return db.DayEmailSlotEvening, true
	}
	return "", false
}

func digestText(tasks []db.ListOpenTasksForDigestRow, day string, slot db.DayEmailSlot, language *string, appURL string) (string, string) {
	english := language != nil && strings.HasPrefix(*language, "en")
	subject := "Задачи на сегодня — Budget Me"
	intro := "Открытые задачи на " + day + ":"
	late := " (с прошлого дня)"
	more := "Осталось ещё задач: %d."
	footer := "Открыть «День»: " + appURL + "/day\nОтключить или изменить письма: " + appURL + "/settings"
	if slot == db.DayEmailSlotEvening {
		subject = "Вечерняя сводка задач — Budget Me"
	}
	if english {
		subject = "Today's tasks — Budget Me"
		if slot == db.DayEmailSlotEvening {
			subject = "Evening task summary — Budget Me"
		}
		intro = "Open tasks for " + day + ":"
		late = " (overdue)"
		more = "More tasks: %d."
		footer = "Open Day: " + appURL + "/day\nChange or stop emails: " + appURL + "/settings"
	}
	var body strings.Builder
	body.WriteString(intro + "\n\n")
	for i, task := range tasks {
		if i == 30 {
			body.WriteString(fmt.Sprintf(more+"\n", len(tasks)-i))
			break
		}
		body.WriteString("• " + task.Title)
		if db.DateStr(task.PlannedFor) < day {
			body.WriteString(late)
		}
		body.WriteByte('\n')
	}
	body.WriteString("\n" + footer + "\n")
	return subject, body.String()
}
