package insights

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/ai"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/insights/commands"
	"budget_me/backend/internal/insights/queries"
)

// retryAfter throttles regeneration per (user, month). Without it every open of
// the screen fired another call while the text was still missing — the provider
// answered 429 within seconds of testing. One attempt at a time, and after a
// failure we wait rather than hammer.
const retryAfter = 2 * time.Minute

// maxConcurrentGen caps how many advice generations may hit the provider AT ONCE
// from the request path. The per-user retryAfter throttle bounds one user, but a
// crowd opening the screen together (each with changed numbers) could still fire
// a burst and draw a 429. The nightly/worker pass is already serial with a pause
// between calls; this closes the last gap. Kept small — glm-4.5-flash needs only
// seconds, so a handful of slots drains a burst quickly.
const maxConcurrentGen = 3

type Controller struct {
	list     queries.ListHandler
	facts    queries.FactsHandler
	generate commands.GenerateAdviceHandler
	q        *db.Queries
	ai       *ai.Client
	clock    clock.Clock

	// attempts remembers when each (user, month) was last handed to the model.
	mu       sync.Mutex
	attempts map[string]time.Time

	// sem is the global concurrency gate for background generations (buffered to
	// maxConcurrentGen). A full buffer means "skip this one" rather than queueing,
	// so no goroutines pile up under load — see rewriteInBackground.
	sem chan struct{}
}

func NewController(q *db.Queries, clk clock.Clock, client *ai.Client) *Controller {
	return &Controller{
		list:     queries.NewListHandler(q, clk),
		facts:    queries.NewFactsHandler(q),
		generate: commands.NewGenerateAdviceHandler(q, client, clk),
		q:        q,
		ai:       client,
		clock:    clk,
		attempts: make(map[string]time.Time),
		sem:      make(chan struct{}, maxConcurrentGen),
	}
}

// claim reports whether this (user, month) may be sent to the model now — false
// while a generation is in flight or a recent one failed.
func (c *Controller) claim(userID uuid.UUID, month string) bool {
	key := userID.String() + "|" + month
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if last, ok := c.attempts[key]; ok && now.Sub(last) < retryAfter {
		return false
	}

	c.attempts[key] = now

	return true
}

// getInsights — GET /insights?month=&tz=. Отвечает СРАЗУ: карточки-правила, цифры
// за месяц и тот текст совета, что уже лежит в кэше. Модель здесь не зовётся
// никогда — она думает секунды (замерено 3–18 с). Если цифры разошлись с тем, из
// чего написан текст, запускаем перезапись В ФОНЕ, и следующий заход её покажет.
func (c *Controller) getInsights(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()
	userID := httpx.UserIDFromContext(reqCtx)
	month := ctx.QueryParam("month")
	timezone := ctx.QueryParam("tz")

	// The billing period, not the calendar month, is what the user budgets in —
	// their monthStartDay setting decides where it opens. Read it so the cards and
	// the advice both measure spend over the same span the dashboard shows.
	user, err := c.q.GetUserByID(reqCtx, ids.ToPg(userID))
	if err != nil {
		return err
	}

	startDay := int(user.MonthStartDay)

	items, err := c.list.Handle(reqCtx, queries.InsightList{
		UserID:        userID,
		Month:         month,
		Timezone:      timezone,
		MonthStartDay: startDay,
	})
	if err != nil {
		return err
	}

	from, to, monthKey := queries.PeriodBounds(startDay, timezone, c.clock.Now())

	facts, err := c.facts.Handle(reqCtx, userID, monthKey, from, to)
	if err != nil {
		return err
	}

	cached, err := c.q.GetAdvice(reqCtx, db.GetAdviceParams{
		UserID: ids.ToPg(userID),
		Month:  facts.Month,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Перезаписываем, когда текст написан НЕ ИЗ ЭТИХ цифр (или его нет вовсе):
	// иначе совет отставал от экрана на сутки — добавил траты днём, а он до ночи
	// рассуждает о старых. Ночной прогон (cmd/worker) остаётся: он ловит тех, кто
	// экран не открывал. Стоимость держит claim(): одна генерация на (юзера,
	// месяц) раз в retryAfter, а не на каждый заход (без него провайдер отвечал
	// 429). Сам ответ не ждёт модель — она думает секунды, см. rewriteInBackground.
	if commands.Stale(cached, facts) && !facts.Empty() && c.ai.Enabled() &&
		c.claim(userID, facts.Month) {
		c.rewriteInBackground(userID, facts)
	}

	out := insightsResponse{
		Insights:       make([]insightResponse, len(items)),
		OverspendMinor: facts.OverspendMinor,
	}

	for i, it := range items {
		out.Insights[i] = toInsightResponse(it)
	}

	// Показываем и «вчерашний» текст, пока идёт перезапись: устаревшая
	// формулировка полезнее пустой карточки.
	if cached.Text != "" {
		out.Advice = &cached.Text
	}

	return httpx.OK(ctx, out)
}

// rewriteInBackground отвязывает генерацию от запроса, который заметил
// устаревание: тот давно отвечен, пока модель думает.
func (c *Controller) rewriteInBackground(userID uuid.UUID, facts queries.Facts) {
	// Take a global slot before spawning. If all are busy we skip rather than
	// queue — the advice refreshes on a later open or the next worker pass (both
	// cheap: unchanged facts hash the same and cost nothing). This keeps a burst
	// of screen-opens from firing a wall of provider calls at once. The send is
	// non-blocking, so the request goroutine never waits on it.
	select {
	case c.sem <- struct{}{}:
	default:
		slog.Debug("insights: generation slots full, deferring", "user", userID)

		return
	}

	go func() {
		defer func() { <-c.sem }()

		bg, cancel := context.WithTimeout(context.Background(), commands.BackgroundTimeout)
		defer cancel()

		if err := c.generate.Handle(bg, commands.GenerateAdvice{
			UserID: userID,
			Facts:  facts,
		}); err != nil {
			// Нет ключа, провайдер чихнул — экран всё равно с цифрами.
			slog.Warn("insights: advice generation failed", "error", err)
		}
	}()
}
