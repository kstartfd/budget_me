package learningpath

import (
	"errors"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/commands"
)

func (c *Controller) assessmentCapabilities(ctx echo.Context) error {
	language := ctx.QueryParam("language")
	if language == "" {
		return apperr.NewInvalid("language is required")
	}
	ctx.Response().Header().Set("Cache-Control", "no-store")
	return httpx.OK(ctx, struct {
		Language  string                  `json:"language"`
		Available []assessment.Capability `json:"available"`
	}{Language: language, Available: assessment.AvailableModes(language, c.assessmentPolicy)})
}

func (c *Controller) assessmentAction(ctx echo.Context) error {
	id, err := idParam(ctx, "assessment")
	if err != nil {
		return err
	}
	action := ctx.Param("action")
	progress, err := c.tests.AssessmentAction(ctx.Request().Context(), userOf(ctx), id, action)
	if err != nil {
		return testError(ctx, err)
	}
	ctx.Response().Header().Set("Cache-Control", "no-store")
	return httpx.OK(ctx, testDTO(progress))
}

func (c *Controller) assessmentResults(ctx echo.Context) error {
	id, err := idParam(ctx, "assessment")
	if err != nil {
		return err
	}
	progress, err := c.tests.Progress(ctx.Request().Context(), userOf(ctx), id)
	if err != nil {
		return err
	}
	if progress.Assessment == nil {
		return apperr.NewNotFound("assessment not found")
	}
	if progress.Assessment.Result == nil {
		return apperr.NewConflict("assessment results are not ready")
	}
	ctx.Response().Header().Set("Cache-Control", "no-store")
	return httpx.OK(ctx, progress.Assessment.Result)
}

func testError(ctx echo.Context, err error) error {
	var limited commands.TestRateError
	if errors.As(err, &limited) {
		return tooManyRequests(ctx, limited.RetryAfter)
	}
	return agentError(err)
}
