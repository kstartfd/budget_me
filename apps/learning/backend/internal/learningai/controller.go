package learningai

import (
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/learningai/commands"
	"learning_app/backend/internal/learningai/queries"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
)

type Controller struct {
	write commands.SaveHandler
	read  queries.SettingsHandler
}

func NewController(q *db.Queries, clk clock.Clock, vault secretbox.Box, client llm.Client) *Controller {
	return &Controller{write: commands.NewSaveHandler(q, clk, vault, client), read: queries.NewSettingsHandler(q)}
}

func (c *Controller) get(ctx echo.Context) error {
	row, err := c.read.Get(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	if row == nil {
		return httpx.OK(ctx, nil)
	}

	return httpx.OK(ctx, settingsDTO(*row))
}

func (c *Controller) save(ctx echo.Context) error {
	var req settingsRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	row, err := c.write.Handle(ctx.Request().Context(), commands.SaveSettings{
		UserID:   httpx.UserIDFromContext(ctx.Request().Context()),
		Provider: llm.Provider(req.Provider),
		Model:    req.Model,
		Level:    req.Level,
		APIKey:   req.APIKey,
	})
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, settingsDTO(row))
}

func (c *Controller) remove(ctx echo.Context) error {
	if err := c.write.Delete(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context())); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}
