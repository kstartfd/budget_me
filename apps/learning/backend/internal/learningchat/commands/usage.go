package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/llm"
)

const (
	usageUser = "user"
	usageKey  = "key"
	// usageKept is how long a day's counters are kept before a turn sweeps them.
	usageKept = 7 * 24 * time.Hour
)

// Usage is what a user, or an API key, may still spend today (UTC). Unlike a
// chat's budget it spans every chat, and it is read before the turn like the
// chat's, so it is soft the same way: by the turns in flight at that moment.
type Usage struct {
	Limit    int64
	Used     int64
	ResetsAt time.Time
}

func (u Usage) Exhausted() bool { return u.Used >= u.Limit }

// After is the usage with a turn costing `spent` counted.
func (u Usage) After(spent int) Usage {
	u.Used += int64(spent)

	return u
}

// Daily is today's usage of a user and of the API key they chat with.
type Daily struct {
	User Usage
	Key  Usage
}

func dayStart(now time.Time) time.Time {
	y, m, d := now.UTC().Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func pgDay(t time.Time) pgtype.Date { return pgtype.Date{Time: dayStart(t), Valid: true} }

// LoadDaily reads today's counters. An empty keyFingerprint skips the key's,
// for callers that only show the user's own usage and have not opened the key.
func LoadDaily(ctx context.Context, q *db.Queries, userID uuid.UUID, keyFingerprint string, profile llm.ChatProfile, now time.Time) (Daily, error) {
	ends := dayStart(now).Add(24 * time.Hour)
	daily := Daily{
		User: Usage{Limit: int64(profile.UserDayTokens), ResetsAt: ends},
		Key:  Usage{Limit: int64(profile.KeyDayTokens), ResetsAt: ends},
	}

	rows, err := q.ListLearningChatUsage(ctx, db.ListLearningChatUsageParams{Day: pgDay(now), UserSubject: userID.String(), KeySubject: keyFingerprint})
	if err != nil {
		return Daily{}, err
	}

	for _, row := range rows {
		switch row.Scope {
		case usageUser:
			daily.User.Used = row.Tokens
		case usageKey:
			daily.Key.Used = row.Tokens
		}
	}

	return daily, nil
}

// recordUsage adds a finished turn to today's counters of the user and of the
// key, and sweeps the days nobody needs any more.
func recordUsage(ctx context.Context, q *db.Queries, userID uuid.UUID, keyFingerprint string, spent int, now time.Time) error {
	day := pgDay(now)

	for scope, subject := range map[string]string{usageUser: userID.String(), usageKey: keyFingerprint} {
		if err := q.AddLearningChatUsage(ctx, db.AddLearningChatUsageParams{Scope: scope, Subject: subject, Day: day, Tokens: int64(spent)}); err != nil {
			return err
		}
	}

	return q.DeleteOldLearningChatUsage(ctx, pgDay(now.Add(-usageKept)))
}

// DailyError is a question refused because the user, or the API key they use,
// has spent the day's tokens.
type DailyError struct{ ResetsAt time.Time }

func (e *DailyError) Error() string {
	return fmt.Sprintf("the daily chat limit is used up until %s", e.ResetsAt.UTC().Format(time.RFC3339))
}
