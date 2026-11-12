package learningpath

import (
	"context"
	"encoding/hex"
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
	"learning_app/backend/internal/learningaudio"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
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
	practices commands.PracticeHandler
	resets    commands.ResetHandler
	clears    commands.ClearHistoryHandler
	read      queries.ReadHandler
	limit     *ratelimit.Limiter
	audio     learningaudio.Service
	testLimit *ratelimit.Limiter
}

func NewController(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, vault secretbox.Box, agent commands.Completer, prefetch int, audio learningaudio.Service) *Controller {
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
		practices: commands.NewPracticeHandler(pool, q, clk, commands.NewAgent(q, vault, agent)),
		resets:    commands.NewResetHandler(q, clk),
		clears:    commands.NewClearHistoryHandler(q),
		read:      queries.NewReadHandler(q),
		limit:     ratelimit.New(aiRate),
		testLimit: ratelimit.New(testRate),
		audio:     audio,
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

// chooseLevel starts a program at a level the learner has already reached,
// with no test: it answers with the finished placement the program is built
// from, like the end of a test. Nothing is asked of the agent here.
func (c *Controller) chooseLevel(ctx echo.Context) error {
	var req startTestRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	test, err := c.tests.Choose(ctx.Request().Context(), commands.ChooseLevel{
		UserID:              userOf(ctx),
		Language:            req.Language,
		TranslationLanguage: req.TranslationLanguage,
		Level:               req.Level,
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

// deleteTeacher removes a teacher from the list; its program goes to the history.
func (c *Controller) deleteTeacher(ctx echo.Context) error {
	id, err := idParam(ctx, "teacher")
	if err != nil {
		return err
	}

	if err := c.teachers.Delete(ctx.Request().Context(), userOf(ctx), id); err != nil {
		return err
	}

	return httpx.OK(ctx, nil)
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

// clearHistory deletes the closed programs of a language; the running one stays.
func (c *Controller) clearHistory(ctx echo.Context) error {
	deleted, err := c.clears.Handle(ctx.Request().Context(), commands.ClearHistory{UserID: userOf(ctx), Language: ctx.QueryParam("language")})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, map[string]int64{"deleted": deleted})
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

	// The rule is shown only to someone who has that teacher now: a program kept
	// in the history of a teacher they deleted does not hand it out.
	var rule *ruleResponse
	if c.hasRuleBook(ctx, detail.Path.Language) {
		rule = ruleOf(detail)
	}

	return httpx.OK(ctx, topicDetailDTO(detail, rule))
}

// hasRuleBook tells whether the learner has a teacher of this language whose
// pair has a rule book.
func (c *Controller) hasRuleBook(ctx echo.Context, language string) bool {
	teachers, err := c.read.Teachers(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return false
	}

	for _, t := range teachers {
		if t.Teacher.Language == language && len(curriculum.Rules(t.Teacher.Language, t.Teacher.TranslationLanguage)) > 0 {
			return true
		}
	}

	return false
}

// ruleOf finds the rule a topic is about: the one it was made from, or, for a
// topic an agent wrote on its own, the one its title names. It is only given
// for a pair that has a rule book, and a program is only ever of such a pair
// when its teacher is, so no one gets a rule they have no teacher for.
func ruleOf(d queries.TopicDetail) *ruleResponse {
	language, native := d.Path.Language, d.Path.TranslationLanguage
	if !curriculum.Has(language, native) {
		return nil
	}

	var (
		found curriculum.Topic
		ok    bool
	)

	if d.Topic.SourceID != "" {
		found, ok = curriculum.Find(language, native, d.Topic.SourceID)
	}

	if !ok || found.Extra == nil {
		reach := min(placement.Index(placement.Level(d.Path.Level))+1, len(placement.Levels)-1)
		found, ok = curriculum.Match(language, native, d.Topic.Title, placement.Levels[max(reach, 0)])
	}

	if !ok || found.Extra == nil {
		return nil
	}

	rule := ruleDTO(found, language, native)

	return &rule
}

// ruleBook is the pair whose rules the learner may read: the one of a teacher
// they have, and only if that pair has a rule book. Without such a teacher the
// rules are not served.
//
// The screen says whose book it asks for with ?language=: the learner who
// studies English and Polish must get the Polish teacher's book (or none, when
// the pair has no book) on the Polish teacher's screen, never the English one.
// Without it the first teacher whose book has the rule is taken, which is what
// the practice calls on a rule's page rely on. A rule that no book has still
// answers from the first book there is, so the caller finds it missing (404)
// instead of the book being closed (403).
func (c *Controller) ruleBook(ctx echo.Context, ruleID string) (string, string, error) {
	teachers, err := c.read.Teachers(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return "", "", err
	}

	want := ctx.QueryParam("language")

	var firstLanguage, firstNative string

	for _, t := range teachers {
		language, native := t.Teacher.Language, t.Teacher.TranslationLanguage
		if want != "" && language != want {
			continue
		}

		if len(curriculum.Rules(language, native)) == 0 {
			continue
		}

		if ruleID == "" {
			return language, native, nil
		}

		if rule, ok := curriculum.Find(language, native, ruleID); ok && rule.Extra != nil {
			return language, native, nil
		}

		if firstLanguage == "" {
			firstLanguage, firstNative = language, native
		}
	}

	if firstLanguage != "" {
		return firstLanguage, firstNative, nil
	}

	return "", "", apperr.NewForbidden("the rule book is for a teacher whose language it covers")
}

// listRules lists the rules of the learner's rule book, by level.
func (c *Controller) listRules(ctx echo.Context) error {
	language, native, err := c.ruleBook(ctx, "")
	if err != nil {
		return err
	}

	rules := curriculum.Rules(language, native)
	out := make([]ruleRef, len(rules))

	for i, r := range rules {
		out[i] = ruleRefOf(r)
	}

	return httpx.OK(ctx, out)
}

// getRule reads one rule, with the rules it refers to.
func (c *Controller) getRule(ctx echo.Context) error {
	language, native, err := c.ruleBook(ctx, ctx.Param("id"))
	if err != nil {
		return err
	}

	rule, ok := curriculum.Find(language, native, ctx.Param("id"))
	if !ok || rule.Extra == nil {
		return apperr.NewNotFound("rule not found")
	}

	ctx.Response().Header().Set("Cache-Control", "private, max-age=3600")

	return httpx.OK(ctx, ruleDTO(rule, language, native))
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

// exerciseAudio reads aloud the sentence of a listening exercise, or the
// phrase of a speaking one.
func (c *Controller) exerciseAudio(ctx echo.Context) error {
	id, err := idParam(ctx, "exercise")
	if err != nil {
		return err
	}

	language, text, err := c.read.ExerciseSpeech(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}

	return c.speak(ctx, language, text)
}

// wordAudio reads aloud one key word of a topic.
func (c *Controller) wordAudio(ctx echo.Context) error {
	id, err := idParam(ctx, "topic")
	if err != nil {
		return err
	}

	index, err := strconv.Atoi(ctx.Param("index"))
	if err != nil {
		return apperr.NewInvalid("invalid word index")
	}

	language, text, err := c.read.WordSpeech(ctx.Request().Context(), userOf(ctx), id, index)
	if err != nil {
		return err
	}

	return c.speak(ctx, language, text)
}

// speak sends the voice of a text: made once, then served from the cache.
func (c *Controller) speak(ctx echo.Context, language, text string) error {
	audio, err := c.audio.Ensure(ctx.Request().Context(), language, text)
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

// startLesson opens the learner's 30-minute lesson, or hands back the one running.
func (c *Controller) startLesson(ctx echo.Context) error {
	lesson, err := c.lessons.Start(ctx.Request().Context(), userOf(ctx))
	if err != nil {
		return err
	}

	return c.lessonReply(ctx, lesson, false)
}

// listLessons lists the learner's lessons, newest first.
func (c *Controller) listLessons(ctx echo.Context) error {
	list, stats, err := c.lessons.List(ctx.Request().Context(), userOf(ctx))
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
