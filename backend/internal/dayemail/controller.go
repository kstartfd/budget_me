package dayemail

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/resend"
)

const confirmationTTL = 24 * time.Hour

type Controller struct {
	q      *db.Queries
	clock  clock.Clock
	mail   *resend.Client
	appURL string
}

func NewController(q *db.Queries, clk clock.Clock, mail *resend.Client, appURL string) *Controller {
	return &Controller{q: q, clock: clk, mail: mail, appURL: strings.TrimRight(appURL, "/")}
}

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/notifications/day-email", c.get)
	protected.PUT("/notifications/day-email", c.update)
	protected.POST("/notifications/day-email/confirm", c.confirm)
}

type preferenceResponse struct {
	Email      *string `json:"email"`
	Frequency  int     `json:"frequency"`
	Status     string  `json:"status"`
	Configured bool    `json:"configured"`
}

type updateRequest struct {
	Frequency int `json:"frequency"`
}

func (c *Controller) preference(ctx echo.Context, userID uuid.UUID) (preferenceResponse, error) {
	u, err := c.q.GetUserByID(ctx.Request().Context(), ids.ToPg(userID))
	if err != nil {
		return preferenceResponse{}, err
	}
	resp := preferenceResponse{
		Email: db.TextPtr(u.Email), Status: "off", Configured: c.mail.Enabled(),
	}
	if resp.Email == nil {
		return resp, nil
	}
	s, err := c.q.GetDayEmailSubscription(ctx.Request().Context(), ids.ToPg(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return resp, nil
	}
	if err != nil {
		return resp, err
	}
	if s.Email != *resp.Email {
		return resp, nil
	}
	resp.Frequency = frequencyNumber(s.Frequency)
	if s.ConfirmedAt.Valid {
		resp.Status = "active"
	} else {
		resp.Status = "pending"
	}
	return resp, nil
}

func (c *Controller) get(ctx echo.Context) error {
	resp, err := c.preference(ctx, httpx.UserIDFromContext(ctx.Request().Context()))
	if err != nil {
		return err
	}
	return httpx.OK(ctx, resp)
}

func (c *Controller) update(ctx echo.Context) error {
	var req updateRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}
	if req.Frequency < 0 || req.Frequency > 2 {
		return apperr.NewInvalid("frequency must be 0, 1 or 2")
	}
	userID := httpx.UserIDFromContext(ctx.Request().Context())
	if req.Frequency == 0 {
		if err := c.q.DeleteDayEmailSubscription(ctx.Request().Context(), ids.ToPg(userID)); err != nil {
			return err
		}
		return c.get(ctx)
	}
	if !c.mail.Enabled() {
		return apperr.NewInvalid("email delivery is not configured")
	}
	u, err := c.q.GetUserByID(ctx.Request().Context(), ids.ToPg(userID))
	if err != nil {
		return err
	}
	email := db.TextPtr(u.Email)
	if email == nil || *email == "" {
		return apperr.NewInvalid("link an email address first")
	}
	frequency := db.DayEmailFrequencyDaily
	if req.Frequency == 2 {
		frequency = db.DayEmailFrequencyTwice
	}
	now := c.clock.Now()
	_, err = c.q.UpdateConfirmedDayEmailFrequency(ctx.Request().Context(), db.UpdateConfirmedDayEmailFrequencyParams{
		Frequency: frequency, Now: db.Time(now), UserID: ids.ToPg(userID), Email: *email,
	})
	if err == nil {
		return c.get(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// A linked login email is not proof of inbox ownership. Send a one-time
	// confirmation before any task title can be delivered to that address.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(digest[:])
	_, err = c.q.UpsertPendingDayEmailSubscription(ctx.Request().Context(), db.UpsertPendingDayEmailSubscriptionParams{
		UserID: ids.ToPg(userID), Email: *email, Frequency: frequency,
		ConfirmationHash: db.Text(&hash), ConfirmationExpiresAt: db.Time(now.Add(confirmationTTL)),
		Now: db.Time(now),
	})
	if err != nil {
		return err
	}
	link := c.appURL + "/settings?dayEmailToken=" + token
	_, err = c.mail.Send(ctx.Request().Context(), *email,
		"Подтвердите уведомления о задачах — Budget Me",
		"Вы запросили письма с задачами раздела «День». Подтвердите адрес в течение 24 часов:\n\n"+link+"\n\nЕсли это были не вы, просто проигнорируйте письмо.",
		"day-confirm-"+userID.String()+"-"+hash[:16],
	)
	if err != nil {
		return err
	}
	return c.get(ctx)
}

func (c *Controller) confirm(ctx echo.Context) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}
	token := req.Token
	if len(token) != 43 {
		return apperr.NewInvalid("invalid confirmation token")
	}
	digest := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(digest[:])
	_, err := c.q.ConfirmDayEmailSubscription(ctx.Request().Context(), db.ConfirmDayEmailSubscriptionParams{
		Now: db.Time(c.clock.Now()), ConfirmationHash: db.Text(&hash),
		UserID: ids.ToPg(httpx.UserIDFromContext(ctx.Request().Context())),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NewInvalid("confirmation link expired or belongs to another account")
		}
		return err
	}
	return c.get(ctx)
}

func frequencyNumber(value db.DayEmailFrequency) int {
	if value == db.DayEmailFrequencyTwice {
		return 2
	}
	return 1
}
