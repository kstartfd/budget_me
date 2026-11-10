package learningmarks

import (
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
)

type Controller struct{ store Store }

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{store: NewStore(q, clk)}
}

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/chat/marks", c.get)
	protected.PUT("/learning/chat/marks", c.save)
	protected.DELETE("/learning/chat/marks", c.remove)
}

func (c *Controller) get(ctx echo.Context) error {
	marks, err := c.store.Get(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	if marks == nil {
		return httpx.OK(ctx, nil)
	}

	return httpx.OK(ctx, marks)
}

func (c *Controller) save(ctx echo.Context) error {
	var req Marks
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	marks, err := c.store.Save(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), req)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, marks)
}

func (c *Controller) remove(ctx echo.Context) error {
	if err := c.store.Delete(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context())); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}
