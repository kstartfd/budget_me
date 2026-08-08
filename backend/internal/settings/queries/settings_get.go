package queries

import (
	"context"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type SettingsGet struct {
	UserID uuid.UUID
	// Timezone — то, что прислал клиент. Запросы считают по ней и без БД; сюда мы
	// её кладём только чтобы фоновые задачи (cmd/worker) знали, где живёт юзер.
	Timezone string
}

type Settings struct {
	MonthStartDay int
	Currency      string
	Today         string
	// HistoryResetAt — момент, с которого считаются текущие итоги (nil, если
	// сброса не было). Всё, что раньше, остаётся в истории и аналитике, но
	// вперёд не переносится.
	HistoryResetAt *time.Time
}

type GetHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewGetHandler(q *db.Queries, clk clock.Clock) GetHandler {
	return GetHandler{q: q, clock: clk}
}

func (h GetHandler) Handle(ctx context.Context, query SettingsGet) (Settings, error) {
	u, err := h.q.GetUserByID(ctx, ids.ToPg(query.UserID))
	if err != nil {
		return Settings{}, err
	}

	if query.Timezone != "" {
		// Пишем, только если зона реально сменилась (SQL сам это проверяет).
		if err := h.q.RememberTimezone(ctx, db.RememberTimezoneParams{
			ID:       ids.ToPg(query.UserID),
			Timezone: query.Timezone,
		}); err != nil {
			return Settings{}, err
		}
	}

	return Settings{
		MonthStartDay:  int(u.MonthStartDay),
		Currency:       string(u.Currency),
		Today:          h.clock.Now().UTC().Format("2006-01-02"),
		HistoryResetAt: db.TimePtr(u.HistoryResetAt),
	}, nil
}
