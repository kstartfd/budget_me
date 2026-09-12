package lists

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/lists/commands"
	"budget_me/backend/internal/lists/queries"
)

var hexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)

type Controller struct {
	create commands.CreateHandler
	remove commands.DeleteHandler
	list   queries.ListHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create: commands.NewCreateHandler(q, clk),
		remove: commands.NewDeleteHandler(q),
		list:   queries.NewListHandler(q),
	}
}

func (c *Controller) getListList(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	rows, err := c.list.Handle(ctx.Request().Context(), queries.ListLists{UserID: userID})
	if err != nil {
		return err
	}

	out := make([]listResponse, len(rows))
	for i, l := range rows {
		out[i] = toListResponse(l)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) createList(ctx echo.Context) error {
	var req createListRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !hexColor.MatchString(req.Color) {
		return apperr.NewInvalid("color must be a #rrggbb hex value")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	l, err := c.create.Handle(ctx.Request().Context(), commands.CreateList{
		UserID: userID,
		Name:   req.Name,
		Color:  req.Color,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toListResponse(l))
}

func (c *Controller) deleteList(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid list id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteList{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
