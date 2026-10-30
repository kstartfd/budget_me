package learningai

import (
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/learningai/commands"
	"learning_app/backend/internal/learningai/queries"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/ratelimit"
	"learning_app/backend/internal/secretbox"
)

// The extension sends a prompt with the page text around a phrase and a few
// earlier turns of the chat; these keep a request to what that needs.
const (
	maxSystemRunes = 10_000
	maxPromptRunes = 60_000
)

// A learner photographs a page and picks from the result; one picture every few
// seconds, one at a time, is far above that and below a script.
var scanRate = ratelimit.Config{Burst: 4, Every: 15 * time.Second, InFlight: 1}

type Controller struct {
	write    commands.SaveHandler
	read     queries.SettingsHandler
	complete commands.CompleteHandler
	scan     commands.ScanHandler
	limit    *ratelimit.Limiter
}

func NewController(q *db.Queries, clk clock.Clock, vault secretbox.Box, client llm.Client) *Controller {
	return &Controller{
		write:    commands.NewSaveHandler(q, clk, vault, client),
		read:     queries.NewSettingsHandler(q),
		complete: commands.NewCompleteHandler(q, vault, client),
		scan:     commands.NewScanHandler(q, vault, client),
		limit:    ratelimit.New(scanRate),
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
		Teacher:  req.Teacher,
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

// scanImage reads the picture from the multipart form. The type is taken from
// the bytes, not from what the client says it is.
func scanImage(ctx echo.Context) (llm.Image, error) {
	file, err := ctx.FormFile("image")
	if err != nil {
		return llm.Image{}, apperr.NewInvalid("a picture is required")
	}

	if file.Size > maxImageBytes {
		return llm.Image{}, apperr.NewInvalid("the picture must be a JPEG, PNG, GIF or WebP up to 6 MB")
	}

	src, err := file.Open()
	if err != nil {
		return llm.Image{}, err
	}

	defer func() { _ = src.Close() }()

	data, err := io.ReadAll(io.LimitReader(src, maxImageBytes+1))
	if err != nil {
		return llm.Image{}, err
	}

	mediaType := http.DetectContentType(data)
	if len(data) > maxImageBytes || !imageTypes[mediaType] {
		return llm.Image{}, apperr.NewInvalid("the picture must be a JPEG, PNG, GIF or WebP up to 6 MB")
	}

	return llm.Image{MediaType: mediaType, Data: data}, nil
}

// read answers with the words and phrases found in a picture of the learner's.
// Like the rest of the agent endpoints it is session-only: an extension token
// must not be able to spend the key.
func (c *Controller) scanPicture(ctx echo.Context) error {
	topicID, err := uuid.Parse(ctx.FormValue("topicId"))
	if err != nil || topicID == uuid.Nil {
		return apperr.NewInvalid("invalid topic id")
	}

	image, err := scanImage(ctx)
	if err != nil {
		return err
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	// A request that is malformed costs nothing and is not counted; one that
	// reaches the agent spends the key, so it is.
	release, wait, ok := c.limit.Acquire(userID)
	if !ok {
		ctx.Response().Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))

		return httpx.TooManyRequests
	}

	defer release()

	found, err := c.scan.Handle(ctx.Request().Context(), commands.Scan{UserID: userID, TopicID: topicID, Image: image})

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

	return httpx.OK(ctx, scanResponse{Items: found})
}
