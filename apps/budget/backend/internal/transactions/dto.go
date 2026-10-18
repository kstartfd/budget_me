package transactions

import (
	"time"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/transactions/queries"
)

// dto.go — слой «форма JSON» фичи «траты»: НИ строки БД (db.*), НИ доменные
// результаты запросов (queries.*), а ровно то, что видит фронт. Теги `json:"..."`
// задают ИМЕНА полей — они обязаны точь-в-точь совпадать с фронтовыми Zod-схемами
// (camelCase!), иначе парс на клиенте упадёт. PHP-параллель: ключи массива в
// JsonResource::toArray().

// transactionRequest — тело POST /transactions. Момент траты приезжает ОДНИМ
// полем в UTC; часовой пояс — забота клиента (он же его и показывает).
type transactionRequest struct {
	AmountMinor int64  `json:"amountMinor"` // копейки Br, всегда > 0 (знак несёт direction)
	CategoryID  string `json:"categoryId"`  // uuid категории как строка
	Title       string `json:"title"`       // мерчант / заголовок строки
	// SpentAt — МОМЕНТ траты в UTC (RFC3339, напр. "2026-07-13T13:30:00Z").
	// Раньше здесь были «голые» date+time — их бэк читал как UTC, хотя фронт
	// слал местное время: трата в 01:00 по Минску сохранялась как 01:00Z, то
	// есть на 3 часа мимо и «вчерашним» днём у границы суток. Момент однозначен.
	SpentAt   string `json:"spentAt"`
	Direction string `json:"direction"` // "income" | "expense"; пусто → expense
	// Counted — участвует ли трата в расчёте (Остаток, «Свободно»). Указатель, чтобы
	// отличить «не прислали» (старый клиент → true) от явного false.
	Counted *bool `json:"counted"`
}

// counted — значение флага «в расчёте» с дефолтом: поле не прислали (старый клиент) →
// участвует, как и вело себя всегда.
func (r transactionRequest) counted() bool {
	return r.Counted == nil || *r.Counted
}

// transactionResponse — форма ОДНОЙ транзакции в ответе (список = срез таких).
type transactionResponse struct {
	ID          string `json:"id"`          // uuid как строка
	AmountMinor int64  `json:"amountMinor"` // копейки Br (положительная величина в обе стороны)
	CategoryID  string `json:"categoryId"`  // uuid категории как строка
	Title       string `json:"title"`       // мерчант / заголовок строки
	SpentAt     string `json:"spentAt"`     // МОМЕНТ траты в UTC (RFC3339)
	Direction   string `json:"direction"`   // "income" | "expense"
	Counted     bool   `json:"counted"`     // участвует ли в расчёте (false — только в статистике)
}

// summaryResponse — JSON-форма ответа /transactions/summary. Совпадает с
// фронтовым MonthSummary; byCategory несёт deltaPct (сравнение с прошлым
// периодом), CarriedInMinor — неизрасходованный приход, перешедший из прошлого
// периода (0, если prevFrom/prevTo не присланы или сравнивать не с чем).
type summaryResponse struct {
	SpentMinor        int64                     `json:"spentMinor"`        // весь расход — для статистики
	CountedSpentMinor int64                     `json:"countedSpentMinor"` // расход, участвующий в расчёте («Свободно», полоска)
	IncomeMinor       int64                     `json:"incomeMinor"`
	CarriedInMinor    int64                     `json:"carriedInMinor"`
	ByCategory        []summaryCategoryResponse `json:"byCategory"`
}

// totalsResponse — форма ответа /transactions/totals: суммарный расход и приход
// за период под текущим фильтром экрана «Транзакции». Совпадает с фронтовым
// TxTotals.
type totalsResponse struct {
	SpentMinor  int64 `json:"spentMinor"`
	IncomeMinor int64 `json:"incomeMinor"`
}

// categorySpendResponse — одна строка разбивки расхода по категории в JSON
// (эндпоинт /by-category). camelCase-теги совпадают с фронтом.
type categorySpendResponse struct {
	CategoryID string `json:"categoryId"` // uuid категории как строка
	SpentMinor int64  `json:"spentMinor"` // сумма трат в копейках Br
}

// summaryCategoryResponse — строка разбивки в ответе сводки: к categoryId+spentMinor
// добавлен deltaPct (% изменения расхода против прошлого периода). Отдельный тип от
// categorySpendResponse, которому дельта не нужна. Указатель + тег БЕЗ omitempty:
// фронтовый MonthSummary ждёт ключ ВСЕГДА (number | null), nil маршалится в JSON null.
type summaryCategoryResponse struct {
	CategoryID string `json:"categoryId"` // uuid категории как строка
	SpentMinor int64  `json:"spentMinor"` // сумма трат в копейках Br
	DeltaPct   *int32 `json:"deltaPct"`   // % против прошлого периода; nil → null
}

// trendPointResponse — один столбик графика на экране «Графики». Совпадает с
// фронтовым TrendPoint {month, spentMinor}.
type trendPointResponse struct {
	Month      string `json:"month"`      // ключ периода, "YYYY-MM"
	SpentMinor int64  `json:"spentMinor"` // расход за период, копейки Br
}

// toTransactionResponse маппит строку БД (db.Transaction с pgtype-полями) в плоский
// JSON-ответ: разворачиваем pgtype-обёртки в обычные Go-типы, момент отдаём в UTC.
func toTransactionResponse(t db.Transaction) transactionResponse {
	return transactionResponse{
		// t.ID — pgtype.UUID (обёртка «16 байт + флаг Valid»). ids.FromPg достаёт
		// google/uuid.UUID, .String() → каноничная строка.
		ID: ids.FromPg(t.ID).String(),

		// amount_minor — bigint NOT NULL, sqlc сгенерил обычный int64 — берём как есть.
		AmountMinor: t.AmountMinor,

		// category_id — pgtype.UUID и МОЖЕТ быть NULL (ON DELETE SET NULL). Тогда
		// Valid == false и выйдет нулевой uuid "00000000-...". Маппим напрямую.
		CategoryID: ids.FromPg(t.CategoryID).String(),

		// title — text NOT NULL → обычная string.
		Title: t.Title,

		// Отдаём МОМЕНТ в UTC (RFC3339) — раскладывать его на «дату и время» в
		// часовом поясе пользователя должен клиент: только он знает, что 22:00Z
		// это уже завтра в Минске. PHP-аналогия: (new DateTimeImmutable(...))
		// ->setTimezone(new DateTimeZone('UTC'))->format(DATE_RFC3339).
		SpentAt: t.SpentAt.Time.UTC().Format(time.RFC3339),

		// direction — enum tx_direction, sqlc сгенерил db.TxDirection (string).
		Direction: string(t.Direction),

		// counted — boolean NOT NULL → обычный bool.
		Counted: t.Counted,
	}
}

// toCategorySpendResponse маппит доменный CategorySpend (uuid уже развёрнут,
// NULL-бакет уже отфильтрован в queries) в JSON-строку.
func toCategorySpendResponse(cs queries.CategorySpend) categorySpendResponse {
	return categorySpendResponse{
		CategoryID: cs.CategoryID.String(),
		SpentMinor: cs.SpentMinor,
	}
}

func toTrendPointResponse(p queries.TrendPoint) trendPointResponse {
	return trendPointResponse{
		Month:      p.Month,
		SpentMinor: p.SpentMinor,
	}
}

// toSummaryResponse собирает JSON-ответ сводки из доменного SummaryResult. ByCategory
// строим через make(len) — на пустом входе даст [] (не null): res.ByCategory всегда
// не nil (см. Handle). DeltaPct пробрасываем как есть — nil даст JSON null.
func toSummaryResponse(res queries.SummaryResult) summaryResponse {
	byCategory := make([]summaryCategoryResponse, len(res.ByCategory))
	for i, cs := range res.ByCategory {
		byCategory[i] = summaryCategoryResponse{
			CategoryID: cs.CategoryID.String(),
			SpentMinor: cs.SpentMinor,
			DeltaPct:   cs.DeltaPct,
		}
	}

	return summaryResponse{
		SpentMinor:        res.SpentMinor,
		CountedSpentMinor: res.CountedSpentMinor,
		IncomeMinor:       res.IncomeMinor,
		CarriedInMinor:    res.CarriedInMinor,
		ByCategory:        byCategory,
	}
}
