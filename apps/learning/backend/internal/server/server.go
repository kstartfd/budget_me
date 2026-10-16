package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"learning_app/backend/internal/auth"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/learning"
	"learning_app/backend/internal/learningtokens"
	learningtokenqueries "learning_app/backend/internal/learningtokens/queries"
	"learning_app/backend/internal/secretbox"
	"learning_app/backend/internal/token"
)

func New(cfg config.Config, pool *pgxpool.Pool) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = httpx.NewErrorHandler(nil)
	e.Use(echomw.Recover())
	e.Use(httpx.RequestID())
	e.Use(httpx.Locale())
	e.Use(httpx.RequestLogger())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderAuthorization, echo.HeaderContentType, "Idempotency-Key"},
	}))
	health := func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "db_down"})
		}
		return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
	}
	e.GET("/healthz", health)
	e.HEAD("/healthz", health)

	clk := clock.New()
	codec := token.NewCodec(cfg.AuthSecret)
	vault := secretbox.New(cfg.AuthSecret)
	queries := db.New(pool)
	public := e.Group("/api/v1")
	protected := e.Group("/api/v1", httpx.RequireAuth(codec))
	auth.NewController(queries, clk, cfg.TelegramBotToken, codec).Routes(public, protected)
	learningAuth := learningtokenqueries.NewTokenHandler(queries, clk, vault)
	group := e.Group("/api/v1/learning", httpx.RequireLearningAuth(codec, learningAuth.Authenticate), echomw.BodyLimit("64K"))
	learning.NewController(queries, clk).Routes(group)
	learningtokens.NewController(queries, clk, vault).Routes(protected)
	return e
}
