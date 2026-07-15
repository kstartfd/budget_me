package queries

import (
	"context"
	"sync"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/rates/provider"
)

// RateList asks every source for the Br price of one Currency ("USD", "EUR",
// "RUB" — the currencies both providers quote). BYN never reaches here: it is
// the currency the amounts are already in, so the UI shows no conversion.
type RateList struct {
	Currency string
}

type RateSource struct {
	Slug string
	Rate float64
}

type namedProvider struct {
	slug     string
	provider provider.Provider
}

type ListHandler struct {
	sources []namedProvider
}

func NewListHandler(nbrb, belarusbank provider.Provider) ListHandler {
	return ListHandler{sources: []namedProvider{
		{slug: "nbrb", provider: nbrb},
		{slug: "belarusbank", provider: belarusbank},
	}}
}

func (h ListHandler) Handle(ctx context.Context, query RateList) ([]RateSource, error) {
	results := make([]RateSource, len(h.sources))

	var wg sync.WaitGroup
	for i, src := range h.sources {
		wg.Add(1)

		go func(i int, src namedProvider) {
			defer wg.Done()

			rate, err := src.provider.Rate(ctx, query.Currency)
			if err != nil {
				return
			}

			results[i] = RateSource{Slug: src.slug, Rate: rate}
		}(i, src)
	}

	wg.Wait()

	out := make([]RateSource, 0, len(results))

	for _, r := range results {
		if r.Rate > 0 {
			out = append(out, r)
		}
	}

	if len(out) == 0 {
		return nil, apperr.NewInternal("could not fetch exchange rates")
	}

	return out, nil
}
