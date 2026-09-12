package tasks

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/tasks/commands"
	"budget_me/backend/internal/tasks/queries"
)

var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Controller struct {
	create  commands.CreateHandler
	update  commands.UpdateHandler
	setDone commands.SetDoneHandler
	remove  commands.DeleteHandler
	dayList queries.DayListHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create:  commands.NewCreateHandler(q, clk),
		update:  commands.NewUpdateHandler(q, clk),
		setDone: commands.NewSetDoneHandler(q, clk),
		remove:  commands.NewDeleteHandler(q),
		dayList: queries.NewDayListHandler(q),
	}
}

func (c *Controller) getDayTasks(ctx echo.Context) error {
	day := ctx.QueryParam("date")
	if !isoDate.MatchString(day) {
		return apperr.NewInvalid("date must be YYYY-MM-DD")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	tasks, err := c.dayList.Handle(ctx.Request().Context(), queries.DayTasks{
		UserID: userID,
		Day:    day,
	})
	if err != nil {
		return err
	}

	out := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskResponse(t)
	}

	return httpx.OK(ctx, out)
}

func (c *Controller) createTask(ctx echo.Context) error {
	var req createTaskRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !isoDate.MatchString(req.PlannedFor) {
		return apperr.NewInvalid("plannedFor must be YYYY-MM-DD")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	task, err := c.create.Handle(ctx.Request().Context(), commands.CreateTask{
		UserID:     userID,
		Title:      req.Title,
		Note:       req.Note,
		PlannedFor: req.PlannedFor,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toTaskResponse(task))
}

func (c *Controller) updateTask(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid task id")
	}

	var req updateTaskRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !isoDate.MatchString(req.PlannedFor) {
		return apperr.NewInvalid("plannedFor must be YYYY-MM-DD")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	task, err := c.update.Handle(ctx.Request().Context(), commands.UpdateTask{
		UserID:     userID,
		ID:         id,
		Title:      req.Title,
		Note:       req.Note,
		PlannedFor: req.PlannedFor,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toTaskResponse(task))
}

func (c *Controller) setTaskDone(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid task id")
	}

	var req setTaskDoneRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if req.DoneOn != nil && !isoDate.MatchString(*req.DoneOn) {
		return apperr.NewInvalid("doneOn must be YYYY-MM-DD")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	task, err := c.setDone.Handle(ctx.Request().Context(), commands.SetTaskDone{
		UserID: userID,
		ID:     id,
		DoneOn: req.DoneOn,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toTaskResponse(task))
}

func (c *Controller) deleteTask(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid task id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteTask{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
