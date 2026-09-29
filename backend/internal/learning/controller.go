package learning

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/learning/commands"
	"budget_me/backend/internal/learning/queries"
)

type Controller struct {
	topics   commands.TopicHandler
	entries  commands.EntryHandler
	captures commands.CaptureHandler
	read     queries.ReadHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{topics: commands.NewTopicHandler(q, clk), entries: commands.NewEntryHandler(q, clk), captures: commands.NewCaptureHandler(q, clk), read: queries.NewReadHandler(q)}
}

func (c *Controller) listTopics(ctx echo.Context) error {
	rows, err := c.read.Topics(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}

	out := make([]topicResponse, len(rows))
	for i, row := range rows {
		out[i] = topicDTO(row)
	}

	return httpx.OK(ctx, out)
}
func (c *Controller) saveTopic(ctx echo.Context) error {
	var req topicRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	id := uuid.Nil

	if ctx.Param("id") != "" {
		parsed, err := uuid.Parse(ctx.Param("id"))
		if err != nil || parsed == uuid.Nil {
			return apperr.NewInvalid("invalid topic id")
		}

		id = parsed
	}

	row, err := c.topics.Handle(ctx.Request().Context(), commands.SaveTopic{UserID: httpx.UserIDFromContext(ctx.Request().Context()), ID: id, Name: req.Name, Language: req.Language, TranslationLanguage: req.TranslationLanguage})
	if err != nil {
		return err
	}

	if id == uuid.Nil {
		return httpx.Created(ctx, topicDTO(row))
	}

	return httpx.OK(ctx, topicDTO(row))
}
func (c *Controller) deleteTopic(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid topic id")
	}

	if err := c.topics.Delete(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
func (c *Controller) listEntries(ctx echo.Context) error {
	topicID, err := uuid.Parse(ctx.QueryParam("topicId"))
	if err != nil {
		return apperr.NewInvalid("invalid topicId")
	}

	limit, offset := 100, 0
	if s := ctx.QueryParam("limit"); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil || limit < 1 || limit > 200 {
			return apperr.NewInvalid("limit must be 1..200")
		}
	}

	if s := ctx.QueryParam("offset"); s != "" {
		offset, err = strconv.Atoi(s)
		if err != nil || offset < 0 || offset > 2147483647 {
			return apperr.NewInvalid("invalid offset")
		}
	}

	kind, status, search := ctx.QueryParam("kind"), ctx.QueryParam("status"), ctx.QueryParam("search")
	if kind != "" && kind != "word" && kind != "phrase" {
		return apperr.NewInvalid("invalid kind filter")
	}

	if status != "" && status != "inbox" && status != "learning" && status != "known" {
		return apperr.NewInvalid("invalid status filter")
	}

	if len(search) > 4000 {
		return apperr.NewInvalid("search exceeds 4000 bytes")
	}

	rows, err := c.read.Entries(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), topicID, int32(limit), int32(offset), kind, status, search)
	if err != nil {
		return err
	}

	out := make([]entryResponse, len(rows))
	for i, row := range rows {
		out[i] = entryDTO(row)
	}

	return httpx.OK(ctx, out)
}
func (c *Controller) getEntry(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid entry id")
	}

	row, err := c.read.Entry(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id)
	if err != nil {
		return err
	}

	return httpx.OK(ctx, entryDTO(row))
}
func (c *Controller) saveEntry(ctx echo.Context) error {
	var fields commands.EntryFields
	if err := ctx.Bind(&fields); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	id := uuid.Nil

	if ctx.Param("id") != "" {
		parsed, err := uuid.Parse(ctx.Param("id"))
		if err != nil || parsed == uuid.Nil {
			return apperr.NewInvalid("invalid entry id")
		}

		id = parsed
	}

	row, err := c.entries.Handle(ctx.Request().Context(), commands.SaveEntry{UserID: httpx.UserIDFromContext(ctx.Request().Context()), ID: id, Fields: fields})
	if err != nil {
		return err
	}

	if id == uuid.Nil {
		return httpx.Created(ctx, entryDTO(row))
	}

	return httpx.OK(ctx, entryDTO(row))
}
func (c *Controller) deleteEntry(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid entry id")
	}

	if err := c.entries.Delete(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
func (c *Controller) enqueue(ctx echo.Context) error {
	var fields commands.EntryFields
	if err := ctx.Bind(&fields); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	row, err := c.captures.Handle(ctx.Request().Context(), commands.EnqueueCapture{UserID: httpx.UserIDFromContext(ctx.Request().Context()), RequestKey: ctx.Request().Header.Get("Idempotency-Key"), Fields: fields})
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Location", "/api/v1/learning/captures/"+captureDTO(row).ID)

	return httpx.Accepted(ctx, captureDTO(row))
}
func (c *Controller) getCapture(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid capture id")
	}

	row, err := c.read.Capture(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id)
	if err != nil {
		return err
	}

	return httpx.OK(ctx, captureDTO(row))
}
