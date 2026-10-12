package queries

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// A recommendation is the median of what the category actually cost over the
// last historyPeriods closed periods, rounded up to a tidy roundToMinor step.
// Periods where the category saw no spend count as zero, so a one-off splurge
// (1 month out of 3) medians down to zero and is not recommended at all.
const (
	historyPeriods = 3
	roundToMinor   = 5_00
	monthKeyLayout = "2006-01"
)

type BudgetRecommendations struct {
	UserID uuid.UUID
	// Timezone — IANA-зона клиента: периоды режутся по его календарю, не по UTC.
	Timezone string
}

type Recommendation struct {
	CategoryID uuid.UUID
	LimitMinor int64
}

type RecommendationsHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewRecommendationsHandler(q *db.Queries, clk clock.Clock) RecommendationsHandler {
	return RecommendationsHandler{q: q, clock: clk}
}

func (h RecommendationsHandler) Handle(
	ctx context.Context,
	query BudgetRecommendations,
) ([]Recommendation, error) {
	user, err := h.q.GetUserByID(ctx, ids.ToPg(query.UserID))
	if err != nil {
		return nil, err
	}

	startDay := int(user.MonthStartDay)
	loc := location(query.Timezone)
	now := h.clock.Now().In(loc)

	// The period containing today is still running — a half-month of spend would
	// drag every median down, so the window ends where the current period opens.
	anchor := now.AddDate(0, 0, -(startDay - 1))
	current := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
	oldest := current.AddDate(0, -historyPeriods, 0)

	from := time.Date(oldest.Year(), oldest.Month(), startDay, 0, 0, 0, 0, loc)
	to := time.Date(current.Year(), current.Month(), startDay, 0, 0, 0, 0, loc)

	rows, err := h.q.SumSpentByPeriodAndCategory(ctx, db.SumSpentByPeriodAndCategoryParams{
		UserID:      ids.ToPg(query.UserID),
		StartOffset: int32(startDay - 1),
		Tz:          loc.String(),
		FromAt:      db.Time(from),
		ToAt:        db.Time(to),
	})
	if err != nil {
		return nil, err
	}

	periods := make([]string, historyPeriods)
	for i := range historyPeriods {
		periods[i] = oldest.AddDate(0, i, 0).Format(monthKeyLayout)
	}

	spent := make(map[uuid.UUID]map[string]int64)

	for _, row := range rows {
		categoryID := ids.FromPg(row.CategoryID)
		if spent[categoryID] == nil {
			spent[categoryID] = make(map[string]int64, historyPeriods)
		}

		spent[categoryID][row.Period] = row.SpentMinor
	}

	out := make([]Recommendation, 0, len(spent))

	for categoryID, byPeriod := range spent {
		history := make([]int64, historyPeriods)
		for i, period := range periods {
			history[i] = byPeriod[period] // missing period → 0
		}

		limit := roundUp(median(history))
		if limit == 0 {
			continue
		}

		out = append(out, Recommendation{CategoryID: categoryID, LimitMinor: limit})
	}

	return out, nil
}

func median(values []int64) int64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}

	return (sorted[mid-1] + sorted[mid]) / 2
}

func roundUp(minor int64) int64 {
	if minor == 0 {
		return 0
	}

	return (minor + roundToMinor - 1) / roundToMinor * roundToMinor
}

// location — см. transactions/queries: кривой пояс не роняет запрос, а даёт UTC.
func location(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}
