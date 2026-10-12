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

// TransactionByCategory — запрос: разбивка расхода юзера по категориям за период.
// From/To — необязательные границы "YYYY-MM-DD"; пустые заполнит resolveRange.
type TransactionByCategory struct {
	UserID uuid.UUID
	From   string
	To     string
}

// ByCategoryHandler — обработчик запроса. Зависимости: sqlc-запросы и часы (для
// дефолта диапазона в resolveRange).
type ByCategoryHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewByCategoryHandler(q *db.Queries, clk clock.Clock) ByCategoryHandler {
	return ByCategoryHandler{q: q, clock: clk}
}

// Handle резолвит диапазон [from, to), берёт агрегат по категориям (уже
// отсортированный spent_minor DESC в SQL) и разворачивает его в []CategorySpend,
// попутно отбрасывая NULL-бакет (см. toCategorySpends). Возвращаемый срез всегда
// НЕ nil — фронт получит [], а не null.
func (h ByCategoryHandler) Handle(ctx context.Context, query TransactionByCategory) ([]CategorySpend, error) {
	from, to, err := resolveRange(h.clock, query.From, query.To)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.SumSpentByCategoryInRange(ctx, db.SumSpentByCategoryInRangeParams{
		UserID: ids.ToPg(query.UserID),
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
	if err != nil {
		return nil, err
	}

	return toCategorySpends(rows), nil
}
