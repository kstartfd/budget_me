package plans

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/plans/commands"
	"budget_me/backend/internal/plans/queries"
)

var monthKey = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type Controller struct {
	create      commands.CreateHandler
	update      commands.UpdateHandler
	confirm     commands.ConfirmHandler
	remove      commands.DeleteHandler
	list        queries.ListHandler
	suggestions queries.SuggestionsHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create:      commands.NewCreateHandler(q, clk),
		update:      commands.NewUpdateHandler(q, clk),
		confirm:     commands.NewConfirmHandler(q, clk),
		remove:      commands.NewDeleteHandler(q),
		list:        queries.NewListHandler(q),
		suggestions: queries.NewSuggestionsHandler(q),
	}
}

func (c *Controller) getPlanList(ctx echo.Context) error {
	month := ctx.QueryParam("month")
	if !monthKey.MatchString(month) {
		return apperr.NewInvalid("month must be YYYY-MM")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	plans, err := c.list.Handle(ctx.Request().Context(), queries.ListPlans{
		UserID: userID,
		Month:  month,
	})
	if err != nil {
		return err
	}

	out := make([]planResponse, len(plans))
	for i, p := range plans {
		out[i] = toPlanResponse(p)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) getPlanSuggestions(ctx echo.Context) error {
	month := ctx.QueryParam("month")
	if !monthKey.MatchString(month) {
		return apperr.NewInvalid("month must be YYYY-MM")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	suggestions, err := c.suggestions.Handle(ctx.Request().Context(), queries.PlanSuggestions{
		UserID: userID,
		Month:  month,
	})
	if err != nil {
		return err
	}

	out := make([]planSuggestionResponse, len(suggestions))
	for i, s := range suggestions {
		out[i] = toPlanSuggestionResponse(s)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) createPlan(ctx echo.Context) error {
	var req createPlanRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return apperr.NewInvalid("categoryId must be a valid uuid")
	}

	if !monthKey.MatchString(req.Month) {
		return apperr.NewInvalid("month must be YYYY-MM")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	plan, err := c.create.Handle(ctx.Request().Context(), commands.CreatePlan{
		UserID:      userID,
		CategoryID:  categoryID,
		Month:       req.Month,
		Title:       req.Title,
		AmountMinor: req.AmountMinor,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toPlanResponse(plan))
}

func (c *Controller) updatePlan(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid plan id")
	}

	var req updatePlanRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return apperr.NewInvalid("categoryId must be a valid uuid")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	plan, err := c.update.Handle(ctx.Request().Context(), commands.UpdatePlan{
		UserID:      userID,
		ID:          id,
		CategoryID:  categoryID,
		Title:       req.Title,
		AmountMinor: req.AmountMinor,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toPlanResponse(plan))
}

func (c *Controller) confirmPlan(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid plan id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	plan, err := c.confirm.Handle(ctx.Request().Context(), commands.ConfirmPlan{
		UserID: userID,
		ID:     id,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toPlanResponse(plan))
}

func (c *Controller) deletePlan(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid plan id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeletePlan{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
