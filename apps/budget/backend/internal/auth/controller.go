package auth

import (
	"net/mail"
	"strings"
	"time"

	ident "budget_me/account"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/auth/commands"
	"budget_me/backend/internal/auth/queries"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
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
	accounts *ident.Store,
	clk clock.Clock,
	botToken string,
	codec *token.Codec,
	deletionGrace time.Duration,
) *Controller {
	links := accountlink.New(q, accounts, clk)

	return &Controller{
		login:     commands.NewLoginHandler(accounts, links),
		telegram:  commands.NewAuthTelegramHandler(q, accounts, links, clk, botToken),
		linkEmail: commands.NewLinkEmailHandler(q, accounts, links, clk),
		user:      queries.NewGetHandler(q, links),
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

	session, err := c.telegram.Handle(ctx.Request().Context(), commands.AuthTelegram{InitData: req.InitData})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, c.session(session))
}

func (c *Controller) loginUser(ctx echo.Context) error {
	var req loginRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	if !validEmail(req.Email) || req.Password == "" {
		return apperr.NewInvalid("invalid params")
	}

	session, err := c.login.Handle(ctx.Request().Context(), commands.Login{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, c.session(session))
}

func (c *Controller) getMe(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	acc, err := c.user.Handle(ctx.Request().Context(), queries.UserGet{ID: userID})
	if err != nil {
		if db.IsNotFound(err) {
			return commands.ErrUnknownAccount
		}

		return err
	}

	return httpx.OK(ctx, toUserResponse(userID, acc, c.deletionGrace))
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

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	acc, err := c.linkEmail.Handle(ctx.Request().Context(), commands.LinkEmail{
		UserID:   userID,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toUserResponse(userID, acc, c.deletionGrace))
}

func (c *Controller) session(s commands.Session) authResponse {
	return authResponse{
		Token: c.codec.Issue(s.UserID, c.clock.Now(), sessionTTL),
		User:  toUserResponse(s.UserID, s.Account, c.deletionGrace),
	}
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}
