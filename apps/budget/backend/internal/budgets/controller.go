package budgets

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/budgets/commands"
	"budget_me/backend/internal/budgets/queries"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
)

var monthKey = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type Controller struct {
	save            commands.SaveHandler
	remove          commands.DeleteHandler
	list            queries.ListHandler
	recommendations queries.RecommendationsHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		save:            commands.NewSaveHandler(q, clk),
		remove:          commands.NewDeleteHandler(q, clk),
		list:            queries.NewListHandler(q),
		recommendations: queries.NewRecommendationsHandler(q, clk),
	}
}

func (c *Controller) getBudgetRecommendations(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	recs, err := c.recommendations.Handle(ctx.Request().Context(), queries.BudgetRecommendations{
		UserID:   userID,
		Timezone: ctx.QueryParam("tz"),
	})
	if err != nil {
		return err
	}

	out := make([]recommendationResponse, len(recs))
	for i, rec := range recs {
		out[i] = toRecommendationResponse(rec)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) getBudgetList(ctx echo.Context) error {
	month := ctx.QueryParam("month")
	if !monthKey.MatchString(month) {
		return apperr.NewInvalid("month must be YYYY-MM")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	budgets, err := c.list.Handle(ctx.Request().Context(), queries.ListBudgets{
		UserID: userID,
		Month:  month,
	})
	if err != nil {
		return err
	}

	out := make([]budgetResponse, len(budgets))
	for i, b := range budgets {
		out[i] = toBudgetResponse(b)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) saveBudget(ctx echo.Context) error {
	var req upsertBudgetRequest
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

	budget, err := c.save.Handle(ctx.Request().Context(), commands.SaveBudget{
		UserID:     userID,
		CategoryID: categoryID,
		ValidFrom:  req.Month,
		LimitMinor: req.LimitMinor,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, savedBudgetResponse(budget))
}

func (c *Controller) deleteBudget(ctx echo.Context) error {
	categoryID, err := uuid.Parse(ctx.Param("categoryId"))
	if err != nil {
		return apperr.NewInvalid("invalid category id")
	}

	month := ctx.QueryParam("month")
	if !monthKey.MatchString(month) {
		return apperr.NewInvalid("month must be YYYY-MM")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteBudget{
		UserID:     userID,
		CategoryID: categoryID,
		Month:      month,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"categoryId": categoryID.String(), "month": month})
}
