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

// TransactionSummary — запрос: сводка за период для дашборда. From/To —
// необязательные границы "YYYY-MM-DD"; пустые заполнит resolveRange. Роль («это
// запрос») несёт пакет queries, поэтому суффикс Query не нужен.
type TransactionSummary struct {
	UserID uuid.UUID
	From   string
	To     string
}

// SummaryResult — то, что дашборду нужно ОДНИМ вызовом: суммарный расход
// (SumSpentInRange), суммарный приход (SumIncomeInRange) и разбивка расхода по
// категориям (SumSpentByCategoryInRange). Все три агрегата в SQL учитывают
// direction, поэтому приход не течёт в расход и наоборот.
//
// PHP-аналогия: readonly-DTO-«конверт» с тремя полями, который хендлер собирает
// из трёх запросов и отдаёт наверх.
type SummaryResult struct {
	SpentMinor  int64
	IncomeMinor int64
	ByCategory  []CategorySpend
}

// SummaryHandler — обработчик запроса. Держит ровно свои зависимости: sqlc-запросы
// и часы (нужны resolveRange для дефолта диапазона).
type SummaryHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewSummaryHandler(q *db.Queries, clk clock.Clock) SummaryHandler {
	return SummaryHandler{q: q, clock: clk}
}

// Handle резолвит один общий диапазон [from, to) и гоняет по нему три агрегата.
// Все они читают одни и те же границы, так что цифры сходятся между собой. Любая
// ошибка БД пробрасывается наверх как есть (транспорт замапит её в 500).
func (h SummaryHandler) Handle(ctx context.Context, query TransactionSummary) (SummaryResult, error) {
	from, to, err := resolveRange(h.clock, query.From, query.To)
	if err != nil {
		return SummaryResult{}, err
	}

	// Границы конвертим в pgtype один раз и переиспользуем во всех трёх запросах.
	userPg := ids.ToPg(query.UserID)
	fromPg := db.Time(from)
	toPg := db.Time(to)

	spent, err := h.q.SumSpentInRange(ctx, db.SumSpentInRangeParams{
		UserID: userPg,
		FromAt: fromPg,
		ToAt:   toPg,
	})
	if err != nil {
		return SummaryResult{}, err
	}

	income, err := h.q.SumIncomeInRange(ctx, db.SumIncomeInRangeParams{
		UserID: userPg,
		FromAt: fromPg,
		ToAt:   toPg,
	})
	if err != nil {
		return SummaryResult{}, err
	}

	byCategory, err := h.q.SumSpentByCategoryInRange(ctx, db.SumSpentByCategoryInRangeParams{
		UserID: userPg,
		FromAt: fromPg,
		ToAt:   toPg,
	})
	if err != nil {
		return SummaryResult{}, err
	}

	return SummaryResult{
		SpentMinor:  spent,
		IncomeMinor: income,
		ByCategory:  toCategorySpends(byCategory), // NULL-бакет отфильтрован
	}, nil
}
