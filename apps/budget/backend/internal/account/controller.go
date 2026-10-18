package account

import (
	"time"

	ident "budget_me/account"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/account/commands"
	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
)

type Controller struct {
	request commands.RequestDeletionHandler
	cancel  commands.CancelDeletionHandler
	grace   time.Duration
}

func NewController(q *db.Queries, accounts *ident.Store, clk clock.Clock, grace time.Duration) *Controller {
	links := accountlink.New(q, accounts, clk)

	return &Controller{
		request: commands.NewRequestDeletionHandler(q, accounts, links, clk),
		cancel:  commands.NewCancelDeletionHandler(q, accounts, links, clk),
		grace:   grace,
	}
}

func (c *Controller) requestDeletion(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	acc, err := c.request.Handle(ctx.Request().Context(), commands.RequestDeletion{
		UserID: userID,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toDeletionResponse(acc, c.grace))
}

func (c *Controller) cancelDeletion(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	acc, err := c.cancel.Handle(ctx.Request().Context(), commands.CancelDeletion{
		UserID: userID,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toDeletionResponse(acc, c.grace))
}
