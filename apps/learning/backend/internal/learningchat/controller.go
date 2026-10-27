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

func entryID(ctx echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperr.NewInvalid("invalid entry id")
	}

	return id, nil
}

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

// send answers as a server-sent event stream: `delta` events with the pieces of
// the answer, then one `done` with the stored messages, or one `error`. A
// failure before the first piece is an ordinary error response instead, so
// a refused question (no agent, chat busy, bad text) needs no stream parsing.
func (c *Controller) send(ctx echo.Context) error {
	id, err := entryID(ctx)
	if err != nil {
		return err
	}

	var req sendRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	// Before anything else, so a flood costs as little as possible.
	release, wait, ok := c.limit.Acquire(userID)
	if !ok {
		retryAfter(ctx, wait)

		return httpx.TooManyRequests
	}

	defer release()

	out := &eventStream{ctx: ctx}

	err = c.write.Handle(ctx.Request().Context(), commands.Send{UserID: userID, EntryID: id, Text: req.Text}, commands.Stream{
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

// agentError sorts a provider failure, or a chat out of tokens, into the errors
// the API speaks in.
func agentError(ctx echo.Context, err error) error {
	var spent *commands.BudgetError
	if errors.As(err, &spent) {
		retryAfter(ctx, time.Until(spent.ResetsAt))

		// A copy, since the details belong to this one answer.
		refused := *httpx.ChatBudgetExhausted
		refused.Details = []string{spent.ResetsAt.UTC().Format(time.RFC3339)}

		return &refused
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
