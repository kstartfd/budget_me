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

// TransactionList — запрос: за какой период читаем траты юзера. Чистый «пакет
// полей» без поведения (в PHP — readonly-DTO). From/To — необязательные границы
// "YYYY-MM-DD"; пустые заполнит дефолт в Handle. Роль («это запрос») несёт пакет
// queries, поэтому суффикс Query не нужен.
type TransactionList struct {
	UserID uuid.UUID
	From   string
	To     string
}

// ListHandler — обработчик запроса. Держит ровно свои зависимости: sqlc-запросы
// и часы. PHP-аналогия — класс-хендлер с инъекцией через конструктор.
type ListHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewListHandler(q *db.Queries, clk clock.Clock) ListHandler {
	return ListHandler{q: q, clock: clk}
}

// Handle приводит границы диапазона и зовёт sqlc. Приёмник «(h ListHandler)» —
// как $this в PHP-методе, только пишется явно. Ошибка — второй результат.
// Резолв диапазона [from, to) вынесен в общий resolveRange (range.go) — та же
// логика, что у summary/spent/by_category, написана один раз.
func (h ListHandler) Handle(ctx context.Context, query TransactionList) ([]db.Transaction, error) {
	from, to, err := resolveRange(h.clock, query.From, query.To)
	if err != nil {
		return nil, err
	}

	return h.q.ListTransactionsInRange(ctx, db.ListTransactionsInRangeParams{
		UserID: ids.ToPg(query.UserID),
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
}
