package settings

import (
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/settings/commands"
	"budget_me/backend/internal/settings/queries"
)

type Controller struct {
	get    queries.GetHandler
	update commands.UpdateHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		get:    queries.NewGetHandler(q, clk),
		update: commands.NewUpdateHandler(q, clk),
	}
}

func (c *Controller) getSettings(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	s, err := c.get.Handle(ctx.Request().Context(), queries.SettingsGet{
		UserID:   userID,
		Timezone: ctx.QueryParam("tz"),
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, settingsResponse{
		MonthStartDay:  s.MonthStartDay,
		Currency:       s.Currency,
		Today:          s.Today,
		HistoryResetAt: resetAtString(s.HistoryResetAt),
	})
}

func (c *Controller) updateSettings(ctx echo.Context) error {
	var req updateSettingsRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	s, err := c.update.Handle(ctx.Request().Context(), commands.UpdateSettings{
		UserID:        userID,
		MonthStartDay: req.MonthStartDay,
		Currency:      req.Currency,
		ResetHistory:  req.ResetHistory,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, settingsResponse{
		MonthStartDay:  s.MonthStartDay,
		Currency:       s.Currency,
		Today:          s.Today,
		HistoryResetAt: resetAtString(s.HistoryResetAt),
	})
}
