package learningpath

import (
	"errors"
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
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/queries"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/ratelimit"
	"learning_app/backend/internal/secretbox"
)

// aiRate paces the calls that spend the learner's key: a burst of starts, then
// one every few seconds, and never two at once. Generation and checking are
// far slower than a learner asks.
var aiRate = ratelimit.Config{Burst: 5, Every: 4 * time.Second, InFlight: 2}

// agentFailed is an agent that answered but its answer survived no parsing,
// even retried. Distinct from the chat's errors so the tab can say what happened.
var agentFailed = httpx.New(30004, http.StatusBadGateway, "the AI agent's answer could not be read, try again")

type Controller struct {
	tests     commands.TestHandler
	analysis  commands.AnalyzeHandler
	exercises commands.ExerciseHandler
	attempts  commands.AttemptHandler
	topics    commands.TopicHandler
	resets    commands.ResetHandler
	read      queries.ReadHandler
	limit     *ratelimit.Limiter
}

func NewController(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, vault secretbox.Box, agent commands.Completer) *Controller {
	return &Controller{
		tests:     commands.NewTestHandler(q, clk),
		analysis:  commands.NewAnalyzeHandler(pool, q, clk, commands.NewAgent(q, vault, agent)),
		exercises: commands.NewExerciseHandler(pool, q, clk, commands.NewAgent(q, vault, agent)),
		attempts:  commands.NewAttemptHandler(q, clk, commands.NewAgent(q, vault, agent), ratelimit.New(aiRate)),
		topics:    commands.NewTopicHandler(q, clk),
		resets:    commands.NewResetHandler(q, clk),
		read:      queries.NewReadHandler(q),
		limit:     ratelimit.New(aiRate),
	}
}

func userOf(ctx echo.Context) uuid.UUID { return httpx.UserIDFromContext(ctx.Request().Context()) }

func idParam(ctx echo.Context, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperr.NewInvalid("invalid " + what + " id")
	}

	return id, nil
}

// agentError sorts an agent failure into what the caller can do about it:
// re-enter the key, try again later, or give up on this call.
func agentError(err error) error {
	switch {
	case errors.Is(err, llm.ErrRejected):
		return apperr.NewInvalid("the provider rejected your API key: enter it again in settings")
	case errors.Is(err, llm.ErrTemporary):
		return httpx.AgentUnavailable
	case errors.Is(err, commands.ErrUnreadable):
		return agentFailed
	}

	return err
}

// tooManyRequests answers a gated call the limiter refused.
func tooManyRequests(ctx echo.Context, wait time.Duration) error {
	ctx.Response().Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))

	return httpx.TooManyRequests
}

// startTest opens a placement test, or hands back the one that is open.
func (c *Controller) startTest(ctx echo.Context) error {
	var req startTestRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	test, err := c.tests.Start(ctx.Request().Context(), commands.StartTest{
		UserID:              userOf(ctx),
		Kind:                "placement",
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
	})
	if err != nil {
		return err
	}

	p, err := c.tests.Progress(ctx.Request().Context(), userOf(ctx), ids.FromPg(test.ID))
	if err != nil {
		return err
	}

	return httpx.Created(ctx, testDTO(p))
}

// startFinal opens the confirmation test of the active program's level.
func (c *Controller) startFinal(ctx echo.Context) error {
	var req startTestRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	test, err := c.tests.Start(ctx.Request().Context(), commands.StartTest{
		UserID:              userOf(ctx),
		Kind:                "final",
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
	})
	if err != nil {
		return err
	}

	p, err := c.tests.Progress(ctx.Request().Context(), userOf(ctx), ids.FromPg(test.ID))
	if err != nil {
		return err
	}

	return httpx.Created(ctx, testDTO(p))
}

func (c *Controller) testState(ctx echo.Context) error {
	id, err := idParam(ctx, "test")
	if err != nil {
		return err
	}

	p, err := c.tests.Progress(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, testDTO(p))
}

func (c *Controller) answer(ctx echo.Context) error {
	id, err := idParam(ctx, "test")
	if err != nil {
		return err
	}

	var req answersRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	given := make([]commands.Answer, len(req.Answers))
	for i, a := range req.Answers {
		given[i] = commands.Answer{Key: a.Key, Answer: a.Answer}
	}

	p, err := c.tests.Answer(ctx.Request().Context(), userOf(ctx), id, given)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, testDTO(p))
}

// analyze has the learner's agent read the finished test and build the
// program from it.
func (c *Controller) analyze(ctx echo.Context) error {
	id, err := idParam(ctx, "test")
	if err != nil {
		return err
	}

	release, wait, ok := c.limit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	path, topics, err := c.analysis.Handle(ctx.Request().Context(), commands.Analyze{UserID: userOf(ctx), TestID: id})
	if err = agentError(err); err != nil {
		return err
	}

	return httpx.Created(ctx, pathDTO(queries.PathWithTopics{Path: path, Topics: topics}))
}

func (c *Controller) current(ctx echo.Context) error {
	language := ctx.QueryParam("language")
	if language == "" {
		return apperr.NewInvalid("language is required")
	}

	found, err := c.read.Active(ctx.Request().Context(), userOf(ctx), language)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, pathDTO(found))
}

func (c *Controller) history(ctx echo.Context) error {
	found, err := c.read.History(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return err
	}

	out := make([]pathResponse, len(found))
	for i, p := range found {
		out[i] = pathDTO(p)
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, out)
}

// reset archives the active program of a language; the history keeps it.
func (c *Controller) reset(ctx echo.Context) error {
	var req startTestRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if err := c.resets.Handle(ctx.Request().Context(), commands.ResetPath{UserID: userOf(ctx), Language: req.Language}); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
}

func (c *Controller) topic(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	detail, err := c.read.Topic(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, topicDetailDTO(detail))
}

// generate writes (or hands back) the exercise set of a topic.
func (c *Controller) generate(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	release, wait, ok := c.limit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	list, err := c.exercises.Handle(ctx.Request().Context(), commands.GenerateExercises{
		UserID:     userOf(ctx),
		TopicID:    id,
		Regenerate: ctx.QueryParam("regenerate") == "1",
	})
	if err = agentError(err); err != nil {
		return err
	}

	return httpx.OK(ctx, exercisesDTO(list))
}

func (c *Controller) completeTopic(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	topic, err := c.topics.Complete(ctx.Request().Context(), commands.CompleteTopic{UserID: userOf(ctx), TopicID: id})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, topicDTO(topic))
}

// attempt checks one answer; the explanation of a mistake comes from the
// learner's agent and is stored with the attempt. Only the agent calls are
// rate-limited — the objective grading is free and instant.
func (c *Controller) attempt(ctx echo.Context) error {
	id, err := idParam(ctx, "exercise")
	if err != nil {
		return err
	}

	var req attemptRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	result, err := c.attempts.Handle(ctx.Request().Context(), commands.SubmitAttempt{UserID: userOf(ctx), ExerciseID: id, Answer: req.Answer})
	if errors.Is(err, commands.ErrThrottled) {
		return httpx.TooManyRequests
	}

	if err = agentError(err); err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, attemptDTO(result.Attempt))
}
