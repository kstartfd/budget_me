package queries

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
)

// The cards this screen can show. Each one is a statement about the user's OWN
// numbers, and every number in it is carried here — the client only formats.
//
// This replaced a set of rules whose copy lived in the translation files with
// invented figures baked in ("Netflix and Spotify unused for 3 weeks — turning
// them off returns 51 Br"): nothing behind them was ever computed.
const (
	KindOverspend = "overspend" // limit exceeded — by how much
	KindNoLimit   = "noLimit"   // biggest category has no limit set
	KindOnTrack   = "onTrack"   // limits exist and all are respected
)

// suggestShare proposes a limit for a category that has none: what it costs
// today, trimmed slightly — a target that is tight but not absurd.
const (
	suggestNum = 9
	suggestDen = 10
)

// Insight is one card. CategoryID is empty for cards that are not about a single
// category (onTrack).
type Insight struct {
	Kind       string
	Tone       string
	CategoryID uuid.UUID
	SpentMinor int64
	LimitMinor int64
	// SuggestMinor is the limit the "set a limit" card proposes.
	SuggestMinor int64
}

type InsightList struct {
	UserID uuid.UUID
	Month  string
	// Timezone — IANA-зона клиента: границы месяца режутся по его календарю.
	Timezone string
}

type ListHandler struct {
	q     *db.Queries
	clock clock.Clock
	facts FactsHandler
}

func NewListHandler(q *db.Queries, clk clock.Clock) ListHandler {
	return ListHandler{q: q, clock: clk, facts: NewFactsHandler(q)}
}

// Handle builds the cards straight from the month's facts: every category that
// broke its limit, then a nudge to put a limit on the biggest category that has
// none, and — when limits exist and none were broken — one card saying so.
func (h ListHandler) Handle(ctx context.Context, query InsightList) ([]Insight, error) {
	from, to := MonthBounds(query.Month, query.Timezone, h.clock.Now())

	facts, err := h.facts.Handle(ctx, query.UserID, from.Format("2006-01"), from, to)
	if err != nil {
		return nil, err
	}

	if facts.Empty() {
		return []Insight{}, nil
	}

	out := make([]Insight, 0, len(facts.Categories))
	hasLimits := false

	for _, c := range facts.Categories {
		if c.LimitMinor == 0 {
			continue
		}

		hasLimits = true

		if c.SpentMinor > c.LimitMinor {
			out = append(out, Insight{
				Kind:       KindOverspend,
				Tone:       "danger",
				CategoryID: c.CategoryID,
				SpentMinor: c.SpentMinor,
				LimitMinor: c.LimitMinor,
			})
		}
	}

	// Categories are sorted by spend, so the first without a limit is the biggest
	// one running unguarded.
	for _, c := range facts.Categories {
		if c.LimitMinor == 0 {
			out = append(out, Insight{
				Kind:         KindNoLimit,
				Tone:         "info",
				CategoryID:   c.CategoryID,
				SpentMinor:   c.SpentMinor,
				SuggestMinor: roundFraction(c.SpentMinor, suggestNum, suggestDen),
			})

			break
		}
	}

	if hasLimits && len(out) == 0 {
		out = append(out, Insight{Kind: KindOnTrack, Tone: "success"})
	}

	return out, nil
}

func roundFraction(v, num, den int64) int64 {
	return (v*num + den/2) / den
}
