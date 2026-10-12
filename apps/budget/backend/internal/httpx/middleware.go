package httpx

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/i18n"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/token"
)

/* --------------------------------- auth ---------------------------------- */

type userIDKey struct{}

// RequireAuth guards the protected route group: it demands a valid
// "Authorization: Bearer <token>" and puts the user id into the request
// context (UserIDFromContext).
func RequireAuth(codec *token.Codec) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get(echo.HeaderAuthorization)
			raw, ok := strings.CutPrefix(header, "Bearer ")

			if !ok || raw == "" {
				return Unauthorized
			}

			userID, err := codec.Parse(raw, time.Now().UTC())
			if err != nil {
				return Unauthorized
			}

			ctx := context.WithValue(c.Request().Context(), userIDKey{}, userID)

			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// UserIDFromContext returns the authenticated user id set by RequireAuth, or
// uuid.Nil on a public route.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey{}).(uuid.UUID)
	return id
}

/* -------------------------------- locale --------------------------------- */

type localeCtxKey struct{}

// Locale parses Accept-Language once per request and threads the negotiated
// locale through the request context; ErrorHandler reads it to localize.
func Locale() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			loc := i18n.ParseAcceptLanguage(c.Request().Header.Get("Accept-Language"))
			req := c.Request()
			c.SetRequest(req.WithContext(context.WithValue(req.Context(), localeCtxKey{}, loc)))

			return next(c)
		}
	}
}

// LocaleFromContext returns the negotiated locale, defaulting to EN.
func LocaleFromContext(ctx context.Context) i18n.Locale {
	if v, ok := ctx.Value(localeCtxKey{}).(i18n.Locale); ok {
		return v
	}

	return i18n.EN
}

/* ------------------------------ request id ------------------------------- */

type requestIDKey struct{}

// RequestID honours an inbound X-Request-ID or mints one, then threads it
// through the response header and the request context so logs correlate.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = ids.New().String()
			}

			c.Response().Header().Set(echo.HeaderXRequestID, id)
			req := c.Request()
			c.SetRequest(req.WithContext(context.WithValue(req.Context(), requestIDKey{}, id)))

			return next(c)
		}
	}
}

// RequestIDFromContext reads the request id past the echo boundary.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}

	return ""
}

/* -------------------------------- logger --------------------------------- */

// RequestLogger emits one structured slog line per request, carrying the
// request id so logs correlate end-to-end.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				// Let the central error sink set the final status, then log.
				c.Error(err)
			}

			req := c.Request()
			slog.Info("request",
				"method", req.Method,
				"path", req.URL.Path,
				"status", c.Response().Status,
				"latency_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(req.Context()),
				"ip", c.RealIP(),
			)

			// Error already delivered via c.Error; don't return it again.
			return nil
		}
	}
}
