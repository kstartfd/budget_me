package balance

import (
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/balance/commands"
	"budget_me/backend/internal/balance/queries"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
)

type Controller struct {
	get queries.GetHandler
	set commands.SetHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		get: queries.NewGetHandler(q),
		set: commands.NewSetHandler(q, clk),
	}
}

func (c *Controller) getBalance(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	res, err := c.get.Handle(ctx.Request().Context(), queries.GetBalance{UserID: userID})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toBalanceResponse(res))
}

func (c *Controller) setBalance(ctx echo.Context) error {
	var req setBalanceRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if _, err := c.set.Handle(ctx.Request().Context(), commands.SetBalance{
		UserID:       userID,
		BalanceMinor: req.BalanceMinor,
	}); err != nil {
		return err
	}

	// Re-read rather than echo the input: the anchor was just set to now, so the
	// balance equals it — but going through the query keeps one definition of
	// "what the balance is".
	res, err := c.get.Handle(ctx.Request().Context(), queries.GetBalance{UserID: userID})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toBalanceResponse(res))
}
