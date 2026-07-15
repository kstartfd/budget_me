package queries

import (
	// Стандартная библиотека.
	"context"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// Сколько периодов показывает график по умолчанию и в каких границах разрешено
// просить другое число (защита от ?months=9999).
const (
	defaultTrendMonths = 6
	maxTrendMonths     = 24
	monthKeyLayout     = "2006-01"
)

// TransactionTrend — запрос: расход по последним N расчётным периодам.
type TransactionTrend struct {
	UserID uuid.UUID
	Months int
	// Timezone — IANA-зона КЛИЕНТА (он один знает её достоверно). Бэкенд её не
	// хранит: она нужна ровно здесь, чтобы разложить UTC-моменты по месяцам так
	// же, как их видит пользователь. Пусто/мусор → UTC.
	Timezone string
}

// TrendPoint — один столбик графика: ключ периода ("2026-07") и расход за него.
type TrendPoint struct {
	Month      string
	SpentMinor int64
}

type TrendHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewTrendHandler(q *db.Queries, clk clock.Clock) TrendHandler {
	return TrendHandler{q: q, clock: clk}
}

// Handle строит НЕПРЕРЫВНЫЙ ряд из N точек, заканчивая текущим периодом. День
// начала месяца берём из настроек пользователя (users.month_start_day, 1..28),
// поэтому «месяц» здесь — расчётный период, а не календарный.
//
// SQL отдаёт только те периоды, где траты были; месяцы-пустышки дошиваем нулями
// здесь — графику нужен ровный ряд, иначе столбики поедут.
//
// PHP-аналогия: собрали array_fill_keys(периоды, 0), затем наложили результат
// GROUP BY из БД и вернули значения по порядку ключей.
func (h TrendHandler) Handle(ctx context.Context, query TransactionTrend) ([]TrendPoint, error) {
	months := query.Months
	if months <= 0 {
		months = defaultTrendMonths
	}

	if months > maxTrendMonths {
		months = maxTrendMonths
	}

	user, err := h.q.GetUserByID(ctx, ids.ToPg(query.UserID))
	if err != nil {
		return nil, err
	}

	startDay := int(user.MonthStartDay)
	loc := location(query.Timezone)
	now := h.clock.Now().In(loc)

	// Тот же сдвиг, что и в SQL: отматываем дату на (startDay-1) дней, и месяц
	// результата — это ключ периода, которому дата принадлежит.
	anchor := now.AddDate(0, 0, -(startDay - 1))
	newest := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
	oldest := newest.AddDate(0, -(months - 1), 0)

	// Начало самого старого периода — startDay-е число его месяца, ПОЛНОЧЬ ПО МЕСТУ
	// (в UTC это, например, 21:00 предыдущего дня). Верхняя граница — «сейчас».
	from := time.Date(oldest.Year(), oldest.Month(), startDay, 0, 0, 0, 0, loc)

	rows, err := h.q.SumSpentByPeriod(ctx, db.SumSpentByPeriodParams{
		UserID:      ids.ToPg(query.UserID),
		StartOffset: int32(startDay - 1),
		Tz:          loc.String(),
		FromAt:      db.Time(from),
		ToAt:        db.Time(now),
	})
	if err != nil {
		return nil, err
	}

	spentByMonth := make(map[string]int64, len(rows))
	for _, row := range rows {
		spentByMonth[row.Period] = row.SpentMinor
	}

	// Ряд от самого старого периода к текущему — фронт рисует столбики слева направо.
	points := make([]TrendPoint, 0, months)

	for i := range months {
		month := oldest.AddDate(0, i, 0).Format(monthKeyLayout)
		points = append(points, TrendPoint{
			Month:      month,
			SpentMinor: spentByMonth[month], // нет трат → нулевой столбик
		})
	}

	return points, nil
}

// location разбирает IANA-имя, присланное клиентом. Мусор и пустая строка дают
// UTC: кривой пояс не должен ронять запрос — в худшем случае месяцы сдвинутся.
func location(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}
