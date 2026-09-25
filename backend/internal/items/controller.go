package items

import (
	"io"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/items/commands"
	"budget_me/backend/internal/items/queries"
	"budget_me/backend/internal/photostore"
)

// maxPhotoUpload caps the raw upload before it is even decoded — photostore
// compresses whatever gets past this down to a few hundred KB, but a phone
// photo can arrive at 10-20 MB, so the cap has to sit above that, not below it.
const maxPhotoUpload = 20 << 20 // 20 MB

type Controller struct {
	create      commands.CreateHandler
	update      commands.UpdateHandler
	remove      commands.DeleteHandler
	uploadPhoto commands.UploadPhotoHandler
	removePhoto commands.RemovePhotoHandler
	list        queries.ListHandler
}

func NewController(q *db.Queries, clk clock.Clock, store photostore.Store) *Controller {
	return &Controller{
		create:      commands.NewCreateHandler(q, clk),
		update:      commands.NewUpdateHandler(q, clk),
		remove:      commands.NewDeleteHandler(q, store),
		uploadPhoto: commands.NewUploadPhotoHandler(q, clk, store),
		removePhoto: commands.NewRemovePhotoHandler(q, clk, store),
		list:        queries.NewListHandler(q),
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

func (c *Controller) uploadItemPhoto(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid item id")
	}

	fh, err := ctx.FormFile("photo")
	if err != nil {
		return apperr.NewInvalid("photo file is required")
	}

	f, err := fh.Open()
	if err != nil {
		return apperr.NewInvalid("could not read photo")
	}

	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, maxPhotoUpload+1))
	if err != nil {
		return apperr.NewInvalid("could not read photo")
	}

	if len(data) > maxPhotoUpload {
		return apperr.NewInvalid("photo is too large")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	item, err := c.uploadPhoto.Handle(ctx.Request().Context(), commands.UploadItemPhoto{
		UserID: userID,
		ID:     id,
		Data:   data,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toItemResponse(item))
}

func (c *Controller) deleteItemPhoto(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid item id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	item, err := c.removePhoto.Handle(ctx.Request().Context(), commands.RemoveItemPhoto{
		UserID: userID,
		ID:     id,
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
