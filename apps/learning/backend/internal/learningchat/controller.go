package learningchat

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/learningchat/commands"
	"learning_app/backend/internal/learningchat/queries"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/ratelimit"
	"learning_app/backend/internal/secretbox"
)

// defaultRate: five questions back to back, then one every four seconds, and at
// most two being answered at once. A person reading each answer asks far
// slower; this only stops a script.
var defaultRate = ratelimit.Config{Burst: 5, Every: 4 * time.Second, InFlight: 2}

type Controller struct {
	write commands.SendHandler
	read  queries.ChatHandler
	limit *ratelimit.Limiter
}

// NewController takes the rate each user may ask at; a zero field falls back
// to the default.
func NewController(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, vault secretbox.Box, client llm.Client, rate ratelimit.Config) *Controller {
	rate.Burst = cmp.Or(rate.Burst, defaultRate.Burst)
	rate.Every = cmp.Or(rate.Every, defaultRate.Every)
	rate.InFlight = cmp.Or(rate.InFlight, defaultRate.InFlight)

	return &Controller{
		write: commands.NewSendHandler(pool, q, clk, vault, client),
		read:  queries.NewChatHandler(q, clk),
		limit: ratelimit.New(rate),
	}
}

func idParam(ctx echo.Context, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperr.NewInvalid("invalid " + what + " id")
	}

	return id, nil
}

func entryID(ctx echo.Context) (uuid.UUID, error) { return idParam(ctx, "entry") }

func topicID(ctx echo.Context) (uuid.UUID, error) { return idParam(ctx, "topic") }

func userOf(ctx echo.Context) uuid.UUID { return httpx.UserIDFromContext(ctx.Request().Context()) }

func (c *Controller) get(ctx echo.Context) error {
	id, err := entryID(ctx)
	if err != nil {
		return err
	}

	chat, err := c.read.Get(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, chatDTO(chat))
}

func (c *Controller) send(ctx echo.Context) error {
	id, err := entryID(ctx)
	if err != nil {
		return err
	}

	return c.stream(ctx, commands.Send{UserID: userOf(ctx), EntryID: id})
}

func (c *Controller) sendTopic(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	return c.stream(ctx, commands.Send{UserID: userOf(ctx), TopicID: id})
}

func (c *Controller) getPathTopic(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	chat, err := c.read.GetPathTopic(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, chatDTO(chat))
}

func (c *Controller) sendPathTopic(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	return c.stream(ctx, commands.Send{UserID: userOf(ctx), PathTopicID: id})
}

func (c *Controller) clearPathTopic(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	if err := c.write.ClearLesson(ctx.Request().Context(), userOf(ctx), id); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}

// stream answers as a server-sent event stream: `delta` events with the pieces of
// the answer, then one `done` with the stored messages, or one `error`. A
// failure before the first piece is an ordinary error response instead, so
// a refused question (no agent, chat busy, bad text) needs no stream parsing.
func (c *Controller) stream(ctx echo.Context, cmd commands.Send) error {
	var req sendRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	cmd.Text = req.Text

	// Before anything else, so a flood costs as little as possible.
	release, wait, ok := c.limit.Acquire(cmd.UserID)
	if !ok {
		retryAfter(ctx, wait)

		return httpx.TooManyRequests
	}

	defer release()

	out := &eventStream{ctx: ctx}

	err := c.write.Handle(ctx.Request().Context(), cmd, commands.Stream{
		Delta: func(piece string) { out.send("delta", deltaEvent{Text: piece}) },
		Done:  func(reply commands.Reply) { out.send("done", doneDTO(reply)) },
	})
	if err = agentError(ctx, err); err == nil {
		return nil
	}

	if !out.started {
		return err
	}

	out.send("error", streamError(err))

	return nil
}

func (c *Controller) getTopic(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	chat, err := c.read.GetTopic(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, chatDTO(chat))
}

// archive is "clear chat" for the free conversation: the dialog is closed and
// kept in the history, and the next message opens a new one.
func (c *Controller) archive(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	if err := c.write.Archive(ctx.Request().Context(), userOf(ctx), id); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}

func (c *Controller) history(ctx echo.Context) error {
	id, err := topicID(ctx)
	if err != nil {
		return err
	}

	dialogs, err := c.read.History(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, dialogsDTO(dialogs))
}

func (c *Controller) dialog(ctx echo.Context) error {
	id, err := idParam(ctx, "dialog")
	if err != nil {
		return err
	}

	transcript, err := c.read.Transcript(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, transcriptDTO(transcript))
}

func (c *Controller) deleteDialog(ctx echo.Context) error {
	id, err := idParam(ctx, "dialog")
	if err != nil {
		return err
	}

	if err := c.write.DeleteDialog(ctx.Request().Context(), userOf(ctx), id); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}

func (c *Controller) clear(ctx echo.Context) error {
	id, err := entryID(ctx)
	if err != nil {
		return err
	}

	if err := c.write.Clear(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}

// retryAfter tells a client how long to wait, in whole seconds and never zero.
func retryAfter(ctx echo.Context, wait time.Duration) {
	ctx.Response().Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))
}

// limitReached is a 429 that says when the limit lifts, in Retry-After and in
// the details.
func limitReached(ctx echo.Context, base *httpx.Error, resetsAt time.Time) *httpx.Error {
	retryAfter(ctx, time.Until(resetsAt))

	// A copy, since the details belong to this one answer.
	refused := *base
	refused.Details = []string{resetsAt.UTC().Format(time.RFC3339)}

	return &refused
}

// agentError sorts a provider failure, or a chat out of tokens, into the errors
// the API speaks in.
func agentError(ctx echo.Context, err error) error {
	var spent *commands.BudgetError
	if errors.As(err, &spent) {
		return limitReached(ctx, httpx.ChatBudgetExhausted, spent.ResetsAt)
	}

	var day *commands.DailyError
	if errors.As(err, &day) {
		return limitReached(ctx, httpx.ChatDailyExhausted, day.ResetsAt)
	}

	switch {
	case errors.Is(err, llm.ErrRejected):
		return apperr.NewInvalid("the provider rejected your API key: enter it again in settings")
	case errors.Is(err, llm.ErrTemporary):
		return httpx.AgentUnavailable
	}

	return err
}

func streamError(err error) errorEvent {
	var apiErr *httpx.Error
	if errors.As(err, &apiErr) {
		return errorEvent{Code: apiErr.Code, Message: apiErr.Msg}
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return errorEvent{Code: httpx.InvalidParamsCode, Message: appErr.Msg}
	}

	return errorEvent{Code: httpx.ServerErrorCode, Message: httpx.ServerError.Msg}
}

// eventStream writes server-sent events and starts the response with the first
// one, so nothing is committed until there is something to say.
type eventStream struct {
	ctx     echo.Context
	started bool
	broken  bool
}

func (s *eventStream) send(event string, data any) {
	if s.broken {
		return
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return
	}

	if !s.started {
		header := s.ctx.Response().Header()
		header.Set(echo.HeaderContentType, "text/event-stream")
		// no-transform keeps a compressing proxy from buffering the stream.
		header.Set("Cache-Control", "no-cache, no-transform")
		header.Set("X-Accel-Buffering", "no")
		s.ctx.Response().WriteHeader(http.StatusOK)

		s.started = true
	}

	// A learner who closed the page stops hearing the answer, but it is still
	// being stored.
	if _, err := fmt.Fprintf(s.ctx.Response(), "event: %s\ndata: %s\n\n", event, raw); err != nil {
		s.broken = true

		return
	}

	s.ctx.Response().Flush()
}
