package rates

import (
	"strings"

	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/rates/provider"
	"budget_me/backend/internal/rates/queries"
)

// Currencies both providers quote against the ruble. BYN is absent on purpose —
// it is what the amounts are already in, so there is nothing to convert.
var quotedCurrencies = map[string]bool{"USD": true, "EUR": true, "RUB": true}

const defaultCurrency = "USD"

type Controller struct {
	list queries.ListHandler
}

func NewController(nbrb, belarusbank provider.Provider) *Controller {
	return &Controller{
		list: queries.NewListHandler(nbrb, belarusbank),
	}
}

func (c *Controller) getRateList(ctx echo.Context) error {
	currency := defaultCurrency
	if raw := ctx.QueryParam("currency"); raw != "" {
		currency = strings.ToUpper(raw)
		if !quotedCurrencies[currency] {
			return apperr.NewInvalid("currency must be one of USD, EUR, RUB")
		}
	}

	sources, err := c.list.Handle(ctx.Request().Context(), queries.RateList{
		Currency: currency,
	})
	if err != nil {
		return err
	}

	out := make([]rateSourceResponse, len(sources))
	for i, s := range sources {
		out[i] = toRateSourceResponse(s)
	}

	return httpx.OK(ctx, out)
}
