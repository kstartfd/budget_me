package items

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/items/commands"
	"budget_me/backend/internal/items/queries"
)

type Controller struct {
	create commands.CreateHandler
	update commands.UpdateHandler
	remove commands.DeleteHandler
	list   queries.ListHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create: commands.NewCreateHandler(q, clk),
		update: commands.NewUpdateHandler(q, clk),
		remove: commands.NewDeleteHandler(q),
		list:   queries.NewListHandler(q),
	}
}

func (c *Controller) getItemList(ctx echo.Context) error {
	listID, err := uuid.Parse(ctx.QueryParam("listId"))
	if err != nil {
		return apperr.NewInvalid("listId must be a valid uuid")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	rows, err := c.list.Handle(ctx.Request().Context(), queries.ListItems{
		UserID: userID,
		ListID: listID,
	})
	if err != nil {
		return err
	}

	out := make([]itemResponse, len(rows))
	for i, it := range rows {
		out[i] = toItemResponse(it)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) createItem(ctx echo.Context) error {
	var req createItemRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	listID, err := uuid.Parse(req.ListID)
	if err != nil {
		return apperr.NewInvalid("listId must be a valid uuid")
	}

	verdict := db.ItemVerdictUnknown

	if req.Verdict != "" {
		v, ok := db.ParseVerdict(req.Verdict)
		if !ok {
			return commands.ErrInvalidVerdict
		}

		verdict = v
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	item, err := c.create.Handle(ctx.Request().Context(), commands.CreateItem{
		UserID:  userID,
		ListID:  listID,
		Title:   req.Title,
		Purpose: req.Purpose,
		Verdict: verdict,
		Rating:  req.Rating,
		Note:    req.Note,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toItemResponse(item))
}

func (c *Controller) updateItem(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid item id")
	}

	var req updateItemRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	verdict := db.ItemVerdictUnknown

	if req.Verdict != "" {
		v, ok := db.ParseVerdict(req.Verdict)
		if !ok {
			return commands.ErrInvalidVerdict
		}

		verdict = v
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	item, err := c.update.Handle(ctx.Request().Context(), commands.UpdateItem{
		UserID:  userID,
		ID:      id,
		Title:   req.Title,
		Purpose: req.Purpose,
		Verdict: verdict,
		Rating:  req.Rating,
		Note:    req.Note,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toItemResponse(item))
}

func (c *Controller) deleteItem(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid item id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteItem{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
