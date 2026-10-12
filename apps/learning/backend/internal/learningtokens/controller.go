package learningtokens

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningtokens/commands"
	"learning_app/backend/internal/learningtokens/queries"
)

type Controller struct {
	write commands.TokenHandler
	read  queries.TokenHandler
}

func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{write: commands.NewTokenHandler(q, clk), read: queries.NewTokenHandler(q, clk)}
}

type tokenResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	Scope      string  `json:"scope"`
	TopicID    *string `json:"topicId"`
	CreatedAt  string  `json:"createdAt"`
	LastUsedAt *string `json:"lastUsedAt"`
	Token      string  `json:"token,omitempty"`
}

func response(row db.LearningToken) tokenResponse {
	r := tokenResponse{ID: ids.FromPg(row.ID).String(), Name: row.Name, Prefix: row.Prefix, Scope: string(row.Scope), CreatedAt: row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")}
	if row.LastUsedAt.Valid {
		s := row.LastUsedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		r.LastUsedAt = &s
	}

	if row.TopicID.Valid {
		id := ids.FromPg(row.TopicID).String()
		r.TopicID = &id
	}

	return r
}
func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/tokens", c.list)
	protected.POST("/learning/tokens", c.create)
	protected.DELETE("/learning/tokens/:id", c.revoke)
}
func (c *Controller) list(ctx echo.Context) error {
	rows, err := c.read.List(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}

	out := make([]tokenResponse, len(rows))
	for i, row := range rows {
		out[i] = response(row)
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, out)
}
func (c *Controller) create(ctx echo.Context) error {
	var req struct {
		Name    string                `json:"name"`
		Scope   db.LearningTokenScope `json:"scope"`
		TopicID string                `json:"topicId"`
	}
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	topicID, err := uuid.Parse(req.TopicID)
	if err != nil {
		return apperr.NewInvalid("invalid topicId")
	}

	row, raw, err := c.write.Handle(ctx.Request().Context(), commands.CreateToken{UserID: httpx.UserIDFromContext(ctx.Request().Context()), Name: req.Name, Scope: req.Scope, TopicID: topicID})
	if err != nil {
		return err
	}

	out := response(row)
	out.Token = raw

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.Created(ctx, out)
}
func (c *Controller) revoke(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid token id")
	}

	if err := c.write.Revoke(ctx.Request().Context(), httpx.UserIDFromContext(ctx.Request().Context()), id); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}
