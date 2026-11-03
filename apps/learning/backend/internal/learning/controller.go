package learning

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/learning/queries"
	"learning_app/backend/internal/learningaudio"
)

type Controller struct {
	topics   commands.TopicHandler
	entries  commands.EntryHandler
	captures commands.CaptureHandler
	read     queries.ReadHandler
	audio    learningaudio.Service
}

func NewController(q *db.Queries, clk clock.Clock, audio learningaudio.Service) *Controller {
	return &Controller{topics: commands.NewTopicHandler(q, clk), entries: commands.NewEntryHandler(q, clk), captures: commands.NewCaptureHandler(q, clk), read: queries.NewReadHandler(q), audio: audio}
}

func (c *Controller) listTopics(ctx echo.Context) error {
	rows, err := c.read.Topics(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}

	// A topic-scoped extension token only gets to see the topic it's bound
	// to, not the account's whole topic list.
	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil {
		filtered := rows[:0]
		for _, row := range rows {
			if ids.FromPg(row.ID) == boundTopic {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
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

	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil && boundTopic != topicID {
		return apperr.NewForbidden("token is scoped to a different topic")
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

	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil && boundTopic != ids.FromPg(row.TopicID) {
		return apperr.NewForbidden("token is scoped to a different topic")
	}

	return httpx.OK(ctx, entryDTO(row))
}
func (c *Controller) getEntryAudio(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid entry id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())
	entry, err := c.read.Entry(ctx.Request().Context(), userID, id)
	if err != nil {
		return err
	}

	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil && boundTopic != ids.FromPg(entry.TopicID) {
		return apperr.NewForbidden("token is scoped to a different topic")
	}

	topic, err := c.read.Topic(ctx.Request().Context(), userID, ids.FromPg(entry.TopicID))
	if err != nil {
		return err
	}

	audio, err := c.audio.Ensure(ctx.Request().Context(), topic.Language, entry.Text)
	if err != nil {
		return err
	}

	file, err := c.audio.Open(ctx.Request().Context(), audio)
	if err != nil {
		return err
	}
	defer file.Close()

	etag := `"` + hex.EncodeToString(audio.ContentHash) + `"`
	ctx.Response().Header().Set("ETag", etag)
	ctx.Response().Header().Set(echo.HeaderCacheControl, "private, max-age=31536000, immutable")
	ctx.Response().Header().Set(echo.HeaderContentType, audio.ContentType)

	if ctx.Request().Header.Get("If-None-Match") == etag {
		return ctx.NoContent(http.StatusNotModified)
	}

	http.ServeContent(ctx.Response(), ctx.Request(), "", time.Time{}, file)

	return nil
}
func (c *Controller) saveEntry(ctx echo.Context) error {
	var fields commands.EntryFields
	if err := ctx.Bind(&fields); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil {
		if fields.TopicID != uuid.Nil && fields.TopicID != boundTopic {
			return apperr.NewForbidden("token is scoped to a different topic")
		}

		fields.TopicID = boundTopic
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

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil {
		row, err := c.read.Entry(ctx.Request().Context(), userID, id)
		if err != nil {
			return err
		}

		if boundTopic != ids.FromPg(row.TopicID) {
			return apperr.NewForbidden("token is scoped to a different topic")
		}
	}

	if err := c.entries.Delete(ctx.Request().Context(), userID, id); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
func (c *Controller) enqueue(ctx echo.Context) error {
	var fields commands.EntryFields
	if err := ctx.Bind(&fields); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	// A topic-scoped extension token never needs to send topicId at all — the
	// endpoint it was handed already implies the topic. If it sends one
	// anyway, it must agree with the token's topic.
	if boundTopic := httpx.LearningTopicFromContext(ctx.Request().Context()); boundTopic != uuid.Nil {
		if fields.TopicID != uuid.Nil && fields.TopicID != boundTopic {
			return apperr.NewForbidden("token is scoped to a different topic")
		}

		fields.TopicID = boundTopic
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
