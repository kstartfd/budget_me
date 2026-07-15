package queries

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// Facts are what the month actually looks like — all computed here, in code, so
// every number the user sees is verifiable. The model never invents one: it only
// gets these and writes a sentence around them.
type Facts struct {
	Month string
	// SpentMinor is this month's total expense; PrevSpentMinor the previous
	// month's, for the "up/down vs last month" line (0 when there is no history).
	SpentMinor     int64
	PrevSpentMinor int64
	IncomeMinor    int64
	// Categories are the user's own expense categories that carry a spend OR a
	// limit this month. Built from their category list, not from the ledger
	// alone: a limit with nothing spent against it never produced a row and so
	// was invisible to the model, which is half of what a limit is for.
	Categories []CategoryFact
	// Untouched names the expense categories with neither spend nor limit — the
	// rest of what the user actually has, so the model knows the list it may
	// speak about and doesn't invent categories.
	Untouched []string
	// OverspendMinor is the sum of how much the limits were exceeded by — the
	// honest version of the old hardcoded "potential saving".
	OverspendMinor int64
}

// LeftMinor is what the screen calls «Осталось»: income minus spend. Negative
// means the month is running on more than it earned.
func (f Facts) LeftMinor() int64 { return f.IncomeMinor - f.SpentMinor }

type CategoryFact struct {
	CategoryID uuid.UUID
	// Name is the human label the user sees: the preset's Russian name (the slug
	// is an internal id — handing "Beauty"/"coffee" to the model made it invent
	// its own translations) or the user's own name for personal categories.
	Name       string
	SpentMinor int64
	// LimitMinor is 0 when the category has no limit this month.
	LimitMinor int64
}

// nearLimitPct is where a category stops being "fine" and starts being "close to
// the limit". The model must never make this call itself — see Verdict.
const nearLimitPct = 80

// UsedPct is how much of the limit is spent, 0 when there is no limit.
func (c CategoryFact) UsedPct() int {
	if c.LimitMinor <= 0 {
		return 0
	}

	return int(c.SpentMinor * 100 / c.LimitMinor)
}

// Verdict is the ONE judgement about a limit, decided here in Go. The model used
// to get bare "потрачено 40.00, лимит 100.00" pairs while being told not to do
// arithmetic — so it guessed, and guessed wrong ("кофе близко к лимиту" at 40%
// spent). Now the verdict is a fact like any other number.
func (c CategoryFact) Verdict() string {
	switch {
	case c.LimitMinor <= 0:
		return "лимита нет"
	case c.SpentMinor > c.LimitMinor:
		return fmt.Sprintf("ЛИМИТ ПРЕВЫШЕН на %.2f", float64(c.SpentMinor-c.LimitMinor)/100)
	case c.UsedPct() >= nearLimitPct:
		return fmt.Sprintf("БЛИЗКО К ЛИМИТУ, осталось %.2f", float64(c.LimitMinor-c.SpentMinor)/100)
	default:
		return fmt.Sprintf("в пределах лимита, осталось %.2f", float64(c.LimitMinor-c.SpentMinor)/100)
	}
}

// Empty reports a month with nothing in it — no spend, nothing to advise on.
func (f Facts) Empty() bool { return f.SpentMinor == 0 }

// promptVersion invalidates every cached advice when the prompt changes shape.
// Without it the nightly pass compares hashes, sees the same numbers, and keeps
// serving text written from the OLD prompt — including the wrong limit claims
// this version exists to fix.
const promptVersion = 2

// Hash identifies the numbers the advice was written from. Same hash → the text
// still describes reality and there is no reason to pay the model again.
func (f Facts) Hash() string {
	var b strings.Builder

	fmt.Fprintf(
		&b, "v%d|%s|%d|%d|%d|",
		promptVersion, f.Month, f.SpentMinor, f.PrevSpentMinor, f.IncomeMinor,
	)

	for _, c := range f.Categories {
		fmt.Fprintf(&b, "%s:%d:%d|", c.Name, c.SpentMinor, c.LimitMinor)
	}

	fmt.Fprintf(&b, "untouched:%s|", strings.Join(f.Untouched, ","))

	sum := sha256.Sum256([]byte(b.String()))

	return hex.EncodeToString(sum[:8])
}

// Prompt is the whole input the model gets: plain numbers, no merchant names, no
// ids — nothing that identifies the person behind them. Every judgement it could
// get wrong (over / near / within a limit) is already made here.
func (f Facts) Prompt() string {
	var b strings.Builder

	b.WriteString("Категории пользователя за месяц (суммы в рублях Br):\n")

	for _, c := range f.Categories {
		fmt.Fprintf(&b, "- %s: потрачено %.2f", c.Name, float64(c.SpentMinor)/100)

		if c.LimitMinor > 0 {
			fmt.Fprintf(
				&b, ", лимит %.2f — израсходовано %d%% лимита",
				float64(c.LimitMinor)/100, c.UsedPct(),
			)
		}

		fmt.Fprintf(&b, " (%s)\n", c.Verdict())
	}

	if len(f.Untouched) > 0 {
		fmt.Fprintf(
			&b, "Категории без трат и без лимита: %s\n",
			strings.Join(f.Untouched, ", "),
		)
	}

	fmt.Fprintf(&b, "Всего расходов: %.2f\n", float64(f.SpentMinor)/100)

	if f.PrevSpentMinor > 0 {
		fmt.Fprintf(&b, "В прошлом месяце: %.2f\n", float64(f.PrevSpentMinor)/100)
	}

	if f.IncomeMinor > 0 {
		fmt.Fprintf(&b, "Доход: %.2f\n", float64(f.IncomeMinor)/100)
		fmt.Fprintf(&b, "Осталось (доход минус расходы): %.2f\n", float64(f.LeftMinor())/100)
	}

	// Both said outright, so "всё в порядке" / "перебрал" is a fact the model
	// reads rather than a conclusion it has to reach on its own.
	if f.OverspendMinor > 0 {
		fmt.Fprintf(
			&b, "Лимиты превышены суммарно на %.2f\n",
			float64(f.OverspendMinor)/100,
		)
	} else {
		b.WriteString("Ни один лимит не превышен.\n")
	}

	return b.String()
}

// presetNames are the preset categories as the USER sees them (the frontend
// renders the same labels from `categories.<slug>`). The slug is an internal id;
// handed one, the model translated it on its own — "Beauty" came back as
// «красота» in advice about a category the screen calls «Красота». An unknown
// slug falls through unchanged rather than becoming an empty name.
var presetNames = map[string]string{
	"coffee":    "Кофе",
	"groceries": "Продукты",
	"dining":    "Кафе и рестораны",
	"transport": "Транспорт",
	"fun":       "Развлечения",
	"subs":      "Подписки",
	"salary":    "Зарплата",
	"dividends": "Дивиденды",
}

func presetName(slug string) string {
	if name, ok := presetNames[slug]; ok {
		return name
	}

	return slug
}

// FactsHandler assembles the month's numbers from the ledger and the limits.
type FactsHandler struct {
	q *db.Queries
}

func NewFactsHandler(q *db.Queries) FactsHandler {
	return FactsHandler{q: q}
}

// Handle builds the facts for [from, to) — the caller passes the boundaries so
// the same rules serve any month.
func (h FactsHandler) Handle(
	ctx context.Context,
	userID uuid.UUID,
	month string,
	from, to time.Time,
) (Facts, error) {
	pgUser := ids.ToPg(userID)

	spent, err := h.q.SumSpentInRange(ctx, db.SumSpentInRangeParams{
		UserID: pgUser,
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
	if err != nil {
		return Facts{}, err
	}

	income, err := h.q.SumIncomeInRange(ctx, db.SumIncomeInRangeParams{
		UserID: pgUser,
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
	if err != nil {
		return Facts{}, err
	}

	prevSpent, err := h.q.SumSpentInRange(ctx, db.SumSpentInRangeParams{
		UserID: pgUser,
		FromAt: db.Time(from.AddDate(0, -1, 0)),
		ToAt:   db.Time(from),
	})
	if err != nil {
		return Facts{}, err
	}

	rows, err := h.q.SumSpentByCategoryInRange(ctx, db.SumSpentByCategoryInRangeParams{
		UserID: pgUser,
		FromAt: db.Time(from),
		ToAt:   db.Time(to),
	})
	if err != nil {
		return Facts{}, err
	}

	cats, err := h.q.ListCategoriesForUser(ctx, pgUser)
	if err != nil {
		return Facts{}, err
	}

	budgets, err := h.q.ListBudgetsForMonth(ctx, db.ListBudgetsForMonthParams{
		UserID: pgUser,
		Month:  month,
	})
	if err != nil {
		return Facts{}, err
	}

	spentByID := make(map[uuid.UUID]int64, len(rows))
	for _, r := range rows {
		spentByID[ids.FromPg(r.CategoryID)] = r.SpentMinor
	}

	limitByID := make(map[uuid.UUID]int64, len(budgets))
	for _, b := range budgets {
		limitByID[ids.FromPg(b.CategoryID)] = b.LimitMinor
	}

	facts := Facts{
		Month:          month,
		SpentMinor:     spent,
		PrevSpentMinor: prevSpent,
		IncomeMinor:    income,
		Categories:     make([]CategoryFact, 0, len(cats)),
	}

	// Walk the user's OWN category list (presets + personal), not the ledger: a
	// limit with nothing spent against it produces no ledger row, and used to
	// vanish from the facts entirely.
	for _, c := range cats {
		if c.Direction != db.TxDirectionExpense {
			continue // доход лимитов не имеет; его итог отдельной строкой
		}

		id := ids.FromPg(c.ID)

		name := presetName(c.Slug.String)
		if c.Name.Valid {
			name = c.Name.String
		}

		spentHere, limit := spentByID[id], limitByID[id]

		if spentHere == 0 && limit == 0 {
			facts.Untouched = append(facts.Untouched, name)

			continue
		}

		if limit > 0 && spentHere > limit {
			facts.OverspendMinor += spentHere - limit
		}

		facts.Categories = append(facts.Categories, CategoryFact{
			CategoryID: id,
			Name:       name,
			SpentMinor: spentHere,
			LimitMinor: limit,
		})
	}

	// Biggest spend first — both the screen and the prompt read top-down.
	sort.Slice(facts.Categories, func(i, j int) bool {
		return facts.Categories[i].SpentMinor > facts.Categories[j].SpentMinor
	})

	return facts, nil
}

// MonthBounds turns "2026-07" into the instants that open and close it IN THE
// USER'S ZONE — the client sends the zone, the backend stores none (see
// migration 0012). Empty/unknown zone falls back to UTC.
func MonthBounds(month, timezone string, now time.Time) (time.Time, time.Time) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	local := now.In(loc)
	year, mon := local.Year(), local.Month()

	if month != "" {
		if p, err := time.Parse("2006-01", month); err == nil {
			year, mon = p.Year(), p.Month()
		}
	}

	from := time.Date(year, mon, 1, 0, 0, 0, 0, loc)

	return from, from.AddDate(0, 1, 0)
}
