package account

import (
	"time"

	"github.com/labstack/echo/v4"

	ident "budget_me/account"
	"learning_app/backend/internal/account/commands"
	"learning_app/backend/internal/accountlink"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
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
	acc, err := c.request.Handle(ctx.Request().Context(), commands.RequestDeletion{
		UserID: httpx.UserIDFromContext(ctx.Request().Context()),
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toDeletionResponse(acc, c.grace))
}

func (c *Controller) cancelDeletion(ctx echo.Context) error {
	acc, err := c.cancel.Handle(ctx.Request().Context(), commands.CancelDeletion{
		UserID: httpx.UserIDFromContext(ctx.Request().Context()),
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toDeletionResponse(acc, c.grace))
}
