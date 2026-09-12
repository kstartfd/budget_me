// Package server assembles the Echo HTTP server and is the composition root:
// cross-cutting middleware, a health check, the /api/v1 route groups and the
// per-feature handlers, all wired here by hand (no DI framework — see
// CLAUDE.md). New feature packages register their routes in New.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"budget_me/backend/internal/account"
	"budget_me/backend/internal/ai"
	"budget_me/backend/internal/alert"
	"budget_me/backend/internal/auth"
	"budget_me/backend/internal/balance"
	"budget_me/backend/internal/budgets"
	"budget_me/backend/internal/categories"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/config"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/insights"
	"budget_me/backend/internal/plans"
	"budget_me/backend/internal/rates"
	"budget_me/backend/internal/rates/provider"
	"budget_me/backend/internal/settings"
	"budget_me/backend/internal/tasks"
	"budget_me/backend/internal/token"
	"budget_me/backend/internal/transactions"
)

// New builds the fully wired Echo server from config + pool.
func New(cfg config.Config, pool *pgxpool.Pool) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// Single error sink: every returned error becomes the standard envelope —
	// and, being the one place a 5xx passes through, the only place alerting has
	// to be wired. No chat configured → the reporter is a no-op.
	alerts := alert.NewReporter(alert.New(cfg.TelegramBotToken, cfg.AlertChatID), cfg.Env)
	e.HTTPErrorHandler = httpx.NewErrorHandler(alerts)

	e.Use(echomw.Recover())
	e.Use(httpx.RequestID())
	e.Use(httpx.Locale())
	e.Use(httpx.RequestLogger())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderAuthorization, echo.HeaderContentType},
		MaxAge:           43200, // 12h preflight cache (seconds)
	}))

	health := func(c echo.Context) error {
		hctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)

		defer cancel()

		if err := pool.Ping(hctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "db_down"})
		}

		return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
	}

	// HEAD as well as GET: uptime monitors probe with HEAD by default (that is
	// how UptimeRobot checks a URL), and a GET-only route answered them 405 —
	// an outage alert every five minutes for a service that was perfectly fine.
	e.GET("/healthz", health)
	e.HEAD("/healthz", health)

	// Shared infrastructure.
	clk := clock.New()
	codec := token.NewCodec(cfg.AuthSecret)
	queries := db.New(pool)

	public := e.Group("/api/v1")
	protected := e.Group("/api/v1", httpx.RequireAuth(codec))

	// Feature handlers — add new domains here.
	auth.NewController(queries, clk, cfg.TelegramBotToken, codec, cfg.AccountDeletionGrace).
		Routes(public, protected)

	account.NewController(queries, clk, cfg.AccountDeletionGrace).
		Routes(public, protected)

	balance.NewController(queries, clk).
		Routes(public, protected)

	insights.NewController(
		queries, clk,
		ai.New(cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL),
	).
		Routes(public, protected)

	rates.NewController(
		provider.NewNBRB(cfg.NBRBBaseURL),
		provider.NewBelarusbank(cfg.BelarusbankBaseURL),
	).
		Routes(public, protected)

	transactions.NewController(queries, clk).
		Routes(public, protected)

	categories.NewController(queries, clk).
		Routes(public, protected)

	budgets.NewController(queries, clk).
		Routes(public, protected)

	plans.NewController(queries, clk).
		Routes(public, protected)

	settings.NewController(queries, clk).
		Routes(public, protected)

	tasks.NewController(queries, clk).
		Routes(public, protected)

	return e
}
