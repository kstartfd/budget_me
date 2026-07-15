package queries

import (
	// Стандартная библиотека.
	"context"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// TransactionSpent — запрос: суммарный расход юзера за период. From/To —
// необязательные границы "YYYY-MM-DD"; пустые заполнит resolveRange.
type TransactionSpent struct {
	UserID uuid.UUID
	From   string
	To     string
}

// SpentHandler — обработчик запроса. Зависимости: sqlc-запросы и часы (для
// дефолта диапазона в resolveRange).
type SpentHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewSpentHandler(q *db.Queries, clk clock.Clock) SpentHandler {
	return SpentHandler{q: q, clock: clk}
}

// Handle резолвит диапазон [from, to) и возвращает сумму расхода в копейках Br.
// При пустой выборке SQL даёт 0 (COALESCE), так что nil-проверок на «нет строк»
// тут не нужно.
func (h SpentHandler) Handle(ctx context.Context, query TransactionSpent) (int64, error) {
	from, to, err := resolveRange(h.clock, query.From, query.To)
	if err != nil {
		return 0, err
	}

	return h.q.SumSpentInRange(ctx, db.SumSpentInRangeParams{
		UserID: ids.ToPg(query.UserID),
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
}
