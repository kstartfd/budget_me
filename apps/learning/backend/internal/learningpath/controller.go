package learningpath

import (
	"context"
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

// testRate paces the level test on its own, so climbing the ladder never eats
// the budget of the program calls. A learner who climbs the whole ladder sends
// one request per level, and each one spends the key twice (the examiner and
// the next batch), so the burst covers the six rungs and a final.
var testRate = ratelimit.Config{Burst: 12, Every: 3 * time.Second, InFlight: 2}

// agentFailed is an agent that answered but its answer survived no parsing,
// even retried. Distinct from the chat's errors so the tab can say what happened.
var agentFailed = httpx.New(30004, http.StatusBadGateway, "the AI agent's answer could not be read, try again")

type Controller struct {
	tests     commands.TestHandler
	analysis  commands.AnalyzeHandler
	builds    commands.BuildHandler
	exercises commands.ExerciseHandler
	attempts  commands.AttemptHandler
	topics    commands.TopicHandler
	teachers  commands.TeacherHandler
	lessons   commands.LessonHandler
	resets    commands.ResetHandler
	read      queries.ReadHandler
	limit     *ratelimit.Limiter
	testLimit *ratelimit.Limiter
}

func NewController(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, vault secretbox.Box, agent commands.Completer, prefetch int) *Controller {
	analysis := commands.NewAnalyzeHandler(pool, q, clk, commands.NewAgent(q, vault, agent))
	exercises := commands.NewExerciseHandler(pool, q, clk, commands.NewAgent(q, vault, agent))

	return &Controller{
		tests:     commands.NewTestHandler(pool, q, clk, commands.NewAgent(q, vault, agent)),
		analysis:  analysis,
		builds:    commands.NewBuildHandler(q, clk, analysis, exercises, prefetch),
		exercises: exercises,
		attempts:  commands.NewAttemptHandler(q, clk, commands.NewAgent(q, vault, agent), ratelimit.New(aiRate)),
		topics:    commands.NewTopicHandler(q, clk),
		teachers:  commands.NewTeacherHandler(q, clk),
		lessons:   commands.NewLessonHandler(q, clk),
		resets:    commands.NewResetHandler(q, clk),
		read:      queries.NewReadHandler(q),
		limit:     ratelimit.New(aiRate),
		testLimit: ratelimit.New(testRate),
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

	// The agent writes the questions, so starting a test spends the learner's
	// key like any other agent call, and fails when it cannot be reached.
	release, wait, ok := c.testLimit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	test, err := c.tests.Start(ctx.Request().Context(), commands.StartTest{
		UserID:              userOf(ctx),
		Kind:                "placement",
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
		StartLevel:          req.Level,
	})
	if err = agentError(err); err != nil {
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

	// The agent writes the questions, so starting a test spends the learner's
	// key like any other agent call, and fails when it cannot be reached.
	release, wait, ok := c.testLimit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	test, err := c.tests.Start(ctx.Request().Context(), commands.StartTest{
		UserID:              userOf(ctx),
		Kind:                "final",
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
	})
	if err = agentError(err); err != nil {
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

	// A full batch goes to the examiner, which is an agent call.
	release, wait, ok := c.testLimit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	p, err := c.tests.Answer(ctx.Request().Context(), userOf(ctx), id, given)
	if err = agentError(err); err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, testDTO(p))
}

// analyze queues the building of the program from a finished test and answers
// at once with the job the screen follows: the agent works in the background,
// so a closed tab or a dropped connection does not lose it. Asking again hands
// the same job back.
func (c *Controller) analyze(ctx echo.Context) error {
	id, err := idParam(ctx, "test")
	if err != nil {
		return err
	}

	// The survey is optional: an empty body means no preferences.
	var req analyzeRequest
	if ctx.Request().ContentLength != 0 {
		if err := ctx.Bind(&req); err != nil {
			return apperr.NewInvalid("invalid body")
		}
	}

	// The slot is held for the whole run, so one learner never builds twice at once.
	release, wait, ok := c.limit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	job, created, err := c.builds.Enqueue(ctx.Request().Context(), commands.BuildProgram{UserID: userOf(ctx), TestID: id, Preferences: commands.Preferences(req)})
	if err != nil {
		release()

		return err
	}

	if created {
		go func() {
			defer release()

			c.builds.Run(job)
		}()
	} else {
		release()
	}

	return httpx.Accepted(ctx, c.jobDTO(ctx, job))
}

// job reports how a build is going, with the program once it exists.
func (c *Controller) job(ctx echo.Context) error {
	id, err := idParam(ctx, "job")
	if err != nil {
		return err
	}

	job, err := c.builds.Job(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, c.jobDTO(ctx, job))
}

// jobDTO renders a job; a finished one carries its program.
func (c *Controller) jobDTO(ctx echo.Context, job db.LearningPathJob) jobResponse {
	out := jobResponse{
		ID:     ids.FromPg(job.ID).String(),
		TestID: ids.FromPg(job.TestID).String(),
		Status: string(job.Status),
		Error:  job.Error,
	}

	if job.Status == db.LearningPathJobStatusDone && job.PathID.Valid {
		if found, err := c.read.PathByID(ctx.Request().Context(), userOf(ctx), ids.FromPg(job.PathID)); err == nil {
			dto := pathDTO(found)
			out.Path = &dto
		}
	}

	return out
}

// listTeachers lists the languages the learner studies, each with a line about
// its running program.
func (c *Controller) listTeachers(ctx echo.Context) error {
	found, err := c.read.Teachers(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return err
	}

	out := make([]teacherResponse, len(found))
	for i, v := range found {
		out[i] = teacherDTO(v)
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, out)
}

// createTeacher adds a language to study, or hands back the one that exists.
func (c *Controller) createTeacher(ctx echo.Context) error {
	var req startTestRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	teacher, err := c.teachers.Create(ctx.Request().Context(), commands.CreateTeacher{
		UserID:              userOf(ctx),
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, teacherDTO(queries.TeacherView{Teacher: teacher}))
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

// lessonReply answers with a lesson and what has been done in it.
func (c *Controller) lessonReply(ctx echo.Context, lesson db.LearningPathLesson, created bool) error {
	stats, err := c.lessons.Stats(ctx.Request().Context(), []db.LearningPathLesson{lesson})
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	body := lessonDTO(lesson, stats[ids.FromPg(lesson.ID)])
	if created {
		return httpx.Created(ctx, body)
	}

	return httpx.OK(ctx, body)
}

// startLesson opens the 30-minute lesson of a topic, or hands back the one running.
func (c *Controller) startLesson(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	lesson, err := c.lessons.Start(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	return c.lessonReply(ctx, lesson, false)
}

// listLessons lists the lessons held on a topic, newest first.
func (c *Controller) listLessons(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	list, stats, err := c.lessons.List(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	out := make([]lessonResponse, len(list))
	for i, l := range list {
		out[i] = lessonDTO(l, stats[ids.FromPg(l.ID)])
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, out)
}

// activeLesson is the running lesson of the learner, or null.
func (c *Controller) activeLesson(ctx echo.Context) error {
	lesson, err := c.lessons.Active(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return err
	}

	if lesson == nil {
		ctx.Response().Header().Set("Cache-Control", "no-store")

		return httpx.OK(ctx, nil)
	}

	return c.lessonReply(ctx, *lesson, false)
}

func (c *Controller) pauseLesson(ctx echo.Context) error {
	return c.changeLesson(ctx, c.lessons.Pause)
}

func (c *Controller) resumeLesson(ctx echo.Context) error {
	return c.changeLesson(ctx, c.lessons.Resume)
}

func (c *Controller) finishLesson(ctx echo.Context) error {
	return c.changeLesson(ctx, c.lessons.Finish)
}

func (c *Controller) changeLesson(ctx echo.Context, change func(context.Context, uuid.UUID, uuid.UUID) (db.LearningPathLesson, error)) error {
	id, err := idParam(ctx, "lesson")
	if err != nil {
		return err
	}

	lesson, err := change(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	return c.lessonReply(ctx, lesson, false)
}
