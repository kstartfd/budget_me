package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/learning/commands"
	"budget_me/backend/internal/token"
)

func main() {
	ctx := context.Background()
	dsn, secret := os.Getenv("DATABASE_URL"), os.Getenv("AUTH_SECRET")
	if os.Getenv("ENV") != "test" || dsn == "" || secret == "" {
		panic("ENV=test, DATABASE_URL and AUTH_SECRET are required; use an isolated test database")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	defer pool.Close()

	q := db.New(pool)
	user := ids.New()

	now := db.Time(time.Now())
	_, err = q.InsertUser(ctx, db.InsertUserParams{ID: ids.ToPg(user), FirstName: "Learning QA", MonthStartDay: 1, Currency: db.CurrencyCodeBYN, Timezone: "UTC", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		panic(err)
	}
	topics := commands.NewTopicHandler(q, clock.New())
	topic, err := topics.Handle(ctx, commands.SaveTopic{UserID: user, Name: "Английский", Language: "en", TranslationLanguage: "ru"})
	if err != nil {
		panic(err)
	}

	entries := commands.NewEntryHandler(q, clock.New())
	fields := []commands.EntryFields{
		{Text: "hit the road", Kind: db.LearningEntryKindPhrase, Transcription: "/hɪt ðə roʊd/", Translation: "отправиться в путь", NativeUsage: "Разговорная фраза: пора уходить или выезжать. Звучит естественно в неформальной беседе.", Example: "It’s getting late. We should hit the road.", Rule: "Идиома: значение выражения не складывается из буквального перевода слов.", Status: db.LearningEntryStatusLearning},
		{Text: "serendipity", Kind: db.LearningEntryKindWord, Transcription: "/ˌserənˈdɪpɪti/", Translation: "счастливая случайность", Status: db.LearningEntryStatusKnown},
		{Text: "I’ll take a rain check", Kind: db.LearningEntryKindPhrase, Translation: "давай в другой раз", Status: db.LearningEntryStatusInbox},
		{Text: "figure out", Kind: db.LearningEntryKindPhrase, Status: db.LearningEntryStatusInbox},
	}

	for _, f := range fields {
		f.TopicID = ids.FromPg(topic.ID)
		_, err = entries.Handle(ctx, commands.SaveEntry{UserID: user, Fields: f})
		if err != nil {
			panic(err)
		}
	}

	out, err := json.Marshal(map[string]string{"userId": user.String(), "topicId": ids.FromPg(topic.ID).String(), "token": token.NewCodec(secret).Issue(user, time.Now(), time.Hour)})
	if err != nil {
		panic(err)
	}

	fixturePath := os.Getenv("E2E_SESSION_FILE")
	if fixturePath == "" {
		fixturePath = "/tmp/budget-learning-fixture.json"
	}

	if err := os.WriteFile(fixturePath, out, 0600); err != nil {
		panic(err)
	}
}
