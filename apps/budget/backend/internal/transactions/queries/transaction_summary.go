package queries

import (
	// Стандартная библиотека.
	"context"
	"math"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// TransactionSummary — запрос: сводка за период для дашборда. From/To —
// необязательные границы RFC3339 (UTC); пустые заполнит resolveRange. PrevFrom/
// PrevTo — границы ПРЕДЫДУЩЕГО периода: тоже RFC3339, но необязательные и
// «мягкие» — нет границ или они кривые, и всё, что от них зависит (дельта по
// категориям, перенос остатка прихода), просто не считается, дашборд из-за
// этого не падает. Роль («это запрос») несёт пакет queries, поэтому суффикс
// Query не нужен.
type TransactionSummary struct {
	UserID   uuid.UUID
	From     string
	To       string
	PrevFrom string
	PrevTo   string
}

// CategorySpendDelta — одна строка «сколько потрачено по категории» в сводке, уже
// с дельтой к прошлому периоду. Отдельна от CategorySpend (у /by-category, где
// дельта не нужна): SpentMinor — расход за текущий период, DeltaPct — процент
// изменения против прошлого, nil когда базы нет (в прошлом периоде по категории 0,
// процент от нуля не определён; на фронте это «нет бейджа»).
//
// PHP-аналогия: маленький readonly-DTO { UuidInterface $categoryId; int $spentMinor;
// ?int $deltaPct }.
type CategorySpendDelta struct {
	CategoryID uuid.UUID
	SpentMinor int64
	DeltaPct   *int32
}

// SummaryResult — то, что дашборду нужно ОДНИМ вызовом: суммарный расход
// (SumSpentInRange), суммарный приход (SumIncomeInRange), разбивка расхода по
// категориям (SumSpentByCategoryInRange) — уже с дельтой к прошлому периоду — и
// CarriedInMinor: неизрасходованный приход прошлого периода (см. carriedInMinor).
// Все агрегаты в SQL учитывают direction, поэтому приход не течёт в расход и
// наоборот.
//
// PHP-аналогия: readonly-DTO-«конверт», который хендлер собирает из нескольких
// запросов и отдаёт наверх.
type SummaryResult struct {
	SpentMinor int64
	// CountedSpentMinor — часть расхода, которая участвует в РАСЧЁТЕ («Свободно»,
	// полоска, перенос): без трат с выключенной галочкой. SpentMinor — весь расход,
	// для статистики.
	CountedSpentMinor int64
	IncomeMinor       int64
	CarriedInMinor    int64
	ByCategory        []CategorySpendDelta
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

// Handle резолвит один общий диапазон [from, to) и гоняет по нему три агрегата (все
// читают одни границы, так что цифры сходятся между собой), затем — при наличии
// окна прошлого периода — считает дельту расхода по каждой категории. Любая ошибка
// БД пробрасывается наверх как есть (транспорт замапит её в 500).
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

	countedSpent, err := h.q.SumCountedSpentInRange(ctx, db.SumCountedSpentInRangeParams{
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

	// Расход по категориям за ПРОШЛЫЙ период — база для дельты. Пустая карта, если
	// окно не прислали (сравнение просто выключено).
	prev, err := h.prevSpendByCategory(ctx, query.UserID, query.PrevFrom, query.PrevTo)
	if err != nil {
		return SummaryResult{}, err
	}

	// Неизрасходованный приход прошлого периода — 0, если окно не прислали
	// (перенос просто выключен, как и дельта выше).
	carried, err := h.carriedIn(ctx, query.UserID, query.PrevFrom, query.PrevTo)
	if err != nil {
		return SummaryResult{}, err
	}

	cur := toCategorySpends(byCategory) // NULL-бакет отфильтрован

	withDelta := make([]CategorySpendDelta, len(cur))
	for i, c := range cur {
		// prev[c.CategoryID] == 0 и для «в прошлом периоде не тратил», и для «окна
		// нет» — в обоих случаях базы нет и deltaPct вернёт nil.
		withDelta[i] = CategorySpendDelta{
			CategoryID: c.CategoryID,
			SpentMinor: c.SpentMinor,
			DeltaPct:   deltaPct(c.SpentMinor, prev[c.CategoryID]),
		}
	}

	return SummaryResult{
		SpentMinor:        spent,
		CountedSpentMinor: countedSpent,
		IncomeMinor:       income,
		CarriedInMinor:    carried,
		ByCategory:        withDelta,
	}, nil
}

// parsePrevWindow разбирает НЕОБЯЗАТЕЛЬНОЕ и «мягкое» окно прошлого периода:
// нет границ, они не RFC3339 или перевёрнуты (to <= from) → ok=false, БЕЗ
// ошибки. Прошлый период — добавка к дашборду (дельта по категориям, перенос
// остатка), а не его условие, поэтому кривой prev не валит запрос (в отличие
// от основного from/to, где resolveRange отдаёт 400). Общий разбор для
// prevSpendByCategory и carriedIn — те же правила «мягкости» в одном месте.
func parsePrevWindow(fromStr, toStr string) (time.Time, time.Time, bool) {
	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, false
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	if !to.After(from) {
		return time.Time{}, time.Time{}, false
	}

	return from, to, true
}

// prevSpendByCategory — расход по категориям за окно прошлого периода, свёрнутый в
// карту categoryID → копейки (база для дельты). Окно отсутствует/кривое → пустая
// карта без ошибки (см. parsePrevWindow). Ошибку БД — другое дело — пробрасываем
// наверх.
func (h SummaryHandler) prevSpendByCategory(ctx context.Context, userID uuid.UUID, fromStr, toStr string) (map[uuid.UUID]int64, error) {
	from, to, ok := parsePrevWindow(fromStr, toStr)
	if !ok {
		return nil, nil
	}

	rows, err := h.q.SumSpentByCategoryInRange(ctx, db.SumSpentByCategoryInRangeParams{
		UserID: ids.ToPg(userID),
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
	if err != nil {
		return nil, err
	}

	out := make(map[uuid.UUID]int64, len(rows))

	for _, row := range rows {
		if !row.CategoryID.Valid { // NULL-бакет удалённой категории — пропускаем
			continue
		}

		out[ids.FromPg(row.CategoryID)] = row.SpentMinor
	}

	return out, nil
}

// carriedIn — сколько неизрасходованного прихода перешло из прошлого периода:
// приход прошлого периода минус его РАСЧЁТНЫЙ расход (без трат с выключенной
// галочкой), за то же окно, что и дельта по категориям. Один хоп назад, БЕЗ цепочки — если и в прошлом периоде что-то
// осталось неизрасходованным, оно уже сидит в Остатке (balance/queries), который
// переносит вообще всё без ограничения на глубину; это — явный бейдж «вот
// сколько перенеслось с прошлого раза», а не второй Остаток. Окно
// отсутствует/кривое → 0, без ошибки (см. parsePrevWindow); ошибку БД
// пробрасываем наверх.
func (h SummaryHandler) carriedIn(ctx context.Context, userID uuid.UUID, fromStr, toStr string) (int64, error) {
	from, to, ok := parsePrevWindow(fromStr, toStr)
	if !ok {
		return 0, nil
	}

	userPg := ids.ToPg(userID)

	// Сброс истории (смена первого дня периода) обрезает окно: то, что было до
	// отсечения, считалось по СТАРЫМ границам периода и вперёд не переносится.
	resetAt, err := h.q.GetHistoryResetAt(ctx, userPg)
	if err != nil {
		return 0, err
	}

	from, ok = carryWindowFrom(from, to, db.TimePtr(resetAt))
	if !ok {
		return 0, nil
	}

	fromPg := db.Time(from)
	toPg := db.Time(to)

	income, err := h.q.SumIncomeInRange(ctx, db.SumIncomeInRangeParams{
		UserID: userPg,
		FromAt: fromPg,
		ToAt:   toPg,
	})
	if err != nil {
		return 0, err
	}

	spent, err := h.q.SumCountedSpentInRange(ctx, db.SumCountedSpentInRangeParams{
		UserID: userPg,
		FromAt: fromPg,
		ToAt:   toPg,
	})
	if err != nil {
		return 0, err
	}

	return carriedInMinor(income, spent), nil
}

// carryWindowFrom обрезает окно прошлого периода [from, to) по точке отсечения
// истории и говорит, осталось ли от него хоть что-то.
//
// Аналогия из PHP: это как `max($from, $cutoff)` — только с ответом «а не пусто
// ли теперь окно вообще».
//
// Отсечения нет (cutoff == nil) → окно как есть. Окно целиком ДО отсечения
// (to <= cutoff) → переносить нечего, ok=false: тот период жил по старым
// границам месяца, и его остаток к новым отношения не имеет. Окно частично
// задевает отсечение → начало подтягивается к нему, хвост после отсечения
// считается как обычно. Граница `to` — исключающая (SQL берёт spent_at < to),
// поэтому to == cutoff это ещё «целиком до».
func carryWindowFrom(from, to time.Time, cutoff *time.Time) (time.Time, bool) {
	if cutoff == nil {
		return from, true
	}

	if !to.After(*cutoff) {
		return time.Time{}, false
	}

	if from.Before(*cutoff) {
		return *cutoff, true
	}

	return from, true
}

// carriedInMinor — арифметика переноса: max(приход − расход, 0) за прошлый
// период. Перетраченный прошлый период (расход > приход) НЕ переносится долгом
// в этот — клэмп в 0, тратить в минус текущий период не должен начинаться с
// чужого перерасхода.
func carriedInMinor(income, spent int64) int64 {
	if income <= spent {
		return 0
	}

	return income - spent
}

// deltaPct — процент изменения расхода категории (cur) против прошлого периода
// (prev), округлённый до целого. База ≤ 0 → nil: процент от нуля не определён, и на
// фронте это «нет бейджа». Знак несёт результат (+ больше, − меньше).
func deltaPct(cur, prev int64) *int32 {
	if prev <= 0 {
		return nil
	}

	p := int32(math.Round(float64(cur-prev) / float64(prev) * 100))

	return &p
}
