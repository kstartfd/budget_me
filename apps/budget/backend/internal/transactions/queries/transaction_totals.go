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

// TransactionTotals — запрос: суммарный расход и приход юзера за период, при
// необходимости сужённые до одной категории. From/To — необязательные границы
// RFC3339 (UTC); пустые заполнит resolveRange. Category — необязательный фильтр
// (uuid.Nil → по всем категориям).
type TransactionTotals struct {
	UserID   uuid.UUID
	From     string
	To       string
	Category uuid.UUID
}

// TotalsResult — то, что нужно шапке экрана «Транзакции»: расход и приход под
// текущим фильтром, оба в копейках Br (положительные величины, знак несёт роль
// поля). Считается на бэке, чтобы цифра была точной независимо от того, как
// список рисуется/пагинируется на клиенте.
type TotalsResult struct {
	SpentMinor  int64
	IncomeMinor int64
}

// TotalsHandler — обработчик запроса. Зависимости: sqlc-запросы и часы (для
// дефолта диапазона в resolveRange).
type TotalsHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTotalsHandler(q *db.Queries, clk clock.Clock) TotalsHandler {
	return TotalsHandler{q: q, clock: clk}
}

// Handle резолвит диапазон [from, to) и возвращает суммы расхода и прихода за
// период. Category == uuid.Nil → фильтра по категории нет (category_id уходит в
// SQL как NULL). При пустой выборке SQL даёт нули (COALESCE), так что nil-проверок
// на «нет строк» тут не нужно.
func (h TotalsHandler) Handle(ctx context.Context, query TransactionTotals) (TotalsResult, error) {
	from, to, err := resolveRange(h.clock, query.From, query.To)
	if err != nil {
		return TotalsResult{}, err
	}

	row, err := h.q.SumTotalsInRange(ctx, db.SumTotalsInRangeParams{
		UserID:     ids.ToPg(query.UserID),
		FromAt:     db.Time(from),
		ToAt:       db.Time(to),
		CategoryID: ids.ToPgOpt(query.Category),
	})
	if err != nil {
		return TotalsResult{}, err
	}

	return TotalsResult{SpentMinor: row.SpentMinor, IncomeMinor: row.IncomeMinor}, nil
}
