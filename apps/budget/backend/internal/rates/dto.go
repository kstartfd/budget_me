package rates

import "budget_me/backend/internal/rates/queries"

type rateSourceResponse struct {
	Slug string  `json:"slug"`
	Rate float64 `json:"rate"`
}

func toRateSourceResponse(s queries.RateSource) rateSourceResponse {
	return rateSourceResponse{Slug: s.Slug, Rate: s.Rate}
}
