package learningai

import (
	"errors"
	"strings"
	"unicode/utf8"

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

// The extension sends a prompt with the page text around a phrase and a few
// earlier turns of the chat; these keep a request to what that needs.
const (
	maxSystemRunes = 10_000
	maxPromptRunes = 60_000
)

type Controller struct {
	write    commands.SaveHandler
	read     queries.SettingsHandler
	complete commands.CompleteHandler
}

func NewController(q *db.Queries, clk clock.Clock, vault secretbox.Box, client llm.Client) *Controller {
	return &Controller{
		write:    commands.NewSaveHandler(q, clk, vault, client),
		read:     queries.NewSettingsHandler(q),
		complete: commands.NewCompleteHandler(q, vault, client),
	}
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

// answer runs a prompt through the user's agent. It is session-only like the
// rest of the agent endpoints: an extension token must not be able to spend the
// key. The key itself never leaves the server.
func (c *Controller) answer(ctx echo.Context) error {
	var req completeRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if strings.TrimSpace(req.Prompt) == "" {
		return apperr.NewInvalid("prompt is required")
	}

	if utf8.RuneCountInString(req.System) > maxSystemRunes || utf8.RuneCountInString(req.Prompt) > maxPromptRunes {
		return apperr.NewInvalid("prompt is too long")
	}

	text, err := c.complete.Handle(ctx.Request().Context(), commands.Complete{
		UserID: httpx.UserIDFromContext(ctx.Request().Context()),
		System: req.System,
		Prompt: req.Prompt,
	})

	switch {
	case errors.Is(err, llm.ErrRejected):
		return agentRejected
	case errors.Is(err, llm.ErrTemporary):
		return agentUnavailable
	case errors.Is(err, commands.ErrAgentFailed):
		return agentFailed
	case err != nil:
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, completeResponse{Text: text})
}
