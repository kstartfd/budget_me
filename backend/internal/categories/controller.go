package categories

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/categories/commands"
	"budget_me/backend/internal/categories/queries"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
)

var hexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)

type Controller struct {
	create commands.CreateHandler
	update commands.UpdateHandler
	remove commands.DeleteHandler
	list   queries.ListHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create: commands.NewCreateHandler(q, clk),
		update: commands.NewUpdateHandler(q),
		remove: commands.NewDeleteHandler(q),
		list:   queries.NewListHandler(q),
	}
}

func (c *Controller) getCategoryList(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	categories, err := c.list.Handle(ctx.Request().Context(), queries.ListCategories{UserID: userID})
	if err != nil {
		return err
	}

	out := make([]categoryResponse, len(categories))
	for i, category := range categories {
		out[i] = toCategoryResponse(category)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) createCategory(ctx echo.Context) error {
	var req createCategoryRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !hexColor.MatchString(req.Color) {
		return apperr.NewInvalid("color must be a #rrggbb hex value")
	}

	direction := db.TxDirectionExpense

	if req.Direction != "" {
		d, ok := db.ParseDirection(req.Direction)
		if !ok {
			return apperr.NewInvalid(`direction must be "income" or "expense"`)
		}

		direction = d
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	cat, err := c.create.Handle(ctx.Request().Context(), commands.CreateCategory{
		UserID:    userID,
		Name:      req.Name,
		Color:     req.Color,
		Direction: direction,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toCategoryResponse(cat))
}

func (c *Controller) updateCategory(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid category id")
	}

	var req updateCategoryRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !hexColor.MatchString(req.Color) {
		return apperr.NewInvalid("color must be a #rrggbb hex value")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	cat, err := c.update.Handle(ctx.Request().Context(), commands.UpdateCategory{
		UserID: userID,
		ID:     id,
		Name:   req.Name,
		Color:  req.Color,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toCategoryResponse(cat))
}

func (c *Controller) deleteCategory(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid category id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteCategory{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
