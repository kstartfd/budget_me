package auth

import (
	"net/mail"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/auth/commands"
	"budget_me/backend/internal/auth/queries"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/token"
)

const sessionTTL = 30 * 24 * time.Hour

type Controller struct {
	login     commands.LoginHandler
	telegram  commands.AuthTelegramHandler
	linkEmail commands.LinkEmailHandler
	user      queries.GetHandler
	codec     *token.Codec
	clock     clock.Clock
	// deletionGrace turns a pending deletion request into the deadline the
	// client shows; the same value gates cmd/worker's sweep.
	deletionGrace time.Duration
}

func NewController(
	q *db.Queries,
	clk clock.Clock,
	botToken string,
	codec *token.Codec,
	deletionGrace time.Duration,
) *Controller {
	return &Controller{
		login:     commands.NewLoginHandler(q),
		telegram:  commands.NewAuthTelegramHandler(q, clk, botToken),
		linkEmail: commands.NewLinkEmailHandler(q, clk),
		user:      queries.NewGetHandler(q),
		codec:     codec,
		clock:     clk,

		deletionGrace: deletionGrace,
	}
}

func (c *Controller) authTelegram(ctx echo.Context) error {
	var req telegramRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if req.InitData == "" {
		return apperr.NewInvalid("init_data is required")
	}

	user, err := c.telegram.Handle(ctx.Request().Context(), commands.AuthTelegram{InitData: req.InitData})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, c.session(user))
}

func (c *Controller) loginUser(ctx echo.Context) error {
	var req loginRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !validEmail(req.Email) || req.Password == "" {
		return apperr.NewInvalid("invalid params")
	}

	user, err := c.login.Handle(ctx.Request().Context(), commands.Login{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, c.session(user))
}

func (c *Controller) getMe(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	user, err := c.user.Handle(ctx.Request().Context(), queries.UserGet{ID: userID})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toUserResponse(user, c.deletionGrace))
}

func (c *Controller) linkEmailIdentity(ctx echo.Context) error {
	var req linkEmailRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !validEmail(req.Email) {
		return apperr.NewInvalid("invalid email")
	}

	if len(req.Password) < 8 {
		return apperr.NewInvalid("password must be at least 8 characters")
	}

	user, err := c.linkEmail.Handle(ctx.Request().Context(), commands.LinkEmail{
		UserID:   httpx.UserIDFromContext(ctx.Request().Context()),
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toUserResponse(user, c.deletionGrace))
}

func (c *Controller) session(user db.User) authResponse {
	return authResponse{
		Token: c.codec.Issue(ids.FromPg(user.ID), c.clock.Now(), sessionTTL),
		User:  toUserResponse(user, c.deletionGrace),
	}
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}
