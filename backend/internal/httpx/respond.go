package httpx

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/i18n"
)

// Body is the standard response envelope: success → code 0; error → the
// business code + message (+ optional details). `data` is always present on
// success (null when the endpoint legitimately has no payload).
type Body struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    any      `json:"data"`
	Details []string `json:"details,omitempty"`
}

func success(c echo.Context, status int, data any) error {
	return c.JSON(status, Body{Code: SuccessCode, Message: "success", Data: data})
}

// OK writes a 200 response wrapping data in the standard envelope.
func OK(c echo.Context, data any) error { return success(c, http.StatusOK, data) }

// Created writes a 201 response wrapping data in the standard envelope.
func Created(c echo.Context, data any) error { return success(c, http.StatusCreated, data) }

func Accepted(c echo.Context, data any) error { return success(c, http.StatusAccepted, data) }

// Reporter is told about every 5xx that leaves the API. Implemented by
// alert.Reporter; nil in tests and on a box with no alert chat configured, which
// is why every call site tolerates a nil.
type Reporter interface {
	Report(method, route string, status int, err error)
}

// ErrorHandler is Echo's single error sink, wired as e.HTTPErrorHandler. It
// renders any returned error into the standard envelope, localized to the
// request locale, by classifying it in a fixed, extensible order:
//
//  1. *Error — a bespoke API error a controller returns to override the
//     mapping; rendered as-is.
//  2. *apperr.Error — a transport-agnostic application error; its Kind is
//     mapped onto the API error model via appErrorStatus.
//  3. *echo.HTTPError — a framework/middleware error (bind, RequireAuth,
//     route 404/405); mapped by fromHTTPError.
//  4. anything else — an unexpected failure rendered as ServerError (500).
func ErrorHandler(err error, c echo.Context) {
	NewErrorHandler(nil)(err, c)
}

// NewErrorHandler is ErrorHandler with a 5xx reporter attached: the ONE place
// every server error already passes through, so alerting needs no per-handler
// plumbing. A nil reporter simply reports nothing.
func NewErrorHandler(reporter Reporter) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		var apiErr *Error

		var appErr *apperr.Error

		switch {
		case errors.As(err, &apiErr):
			// already an api error — explicit controller override
		case errors.As(err, &appErr):
			apiErr = fromAppError(appErr)
		case isHTTPError(err):
			apiErr = fromHTTPError(err)
		default:
			apiErr = ServerError
		}

		status := apiErr.StatusCode()

		if reporter != nil && status >= http.StatusInternalServerError {
			// c.Path() is the ROUTE ("/api/v1/transactions/:id"), so no id from a
			// real request ends up in the alert.
			reporter.Report(c.Request().Method, c.Path(), status, err)
		}

		loc := LocaleFromContext(c.Request().Context())
		_ = c.JSON(status, Body{
			Code:    apiErr.Code,
			Message: i18n.T(loc, apiErr.Msg),
			Details: apiErr.Details,
		})
	}
}

// appErrorStatus maps each apperr.Kind onto the predefined API error that
// carries its business code and HTTP status. Extend this table (and add the
// code constant + predefined error in errors.go) to route a new Kind.
var appErrorStatus = map[apperr.Kind]*Error{
	apperr.Invalid:      InvalidParams, // 400 / 10001
	apperr.Unauthorized: Unauthorized,  // 401 / 20001
	apperr.Forbidden:    Forbidden,     // 403 / 20002
	apperr.NotFound:     NotFound,      // 404 / 10002
	apperr.Conflict:     Conflict,      // 409 / 40001
	apperr.Internal:     ServerError,   // 500 / 10000
}

// fromAppError maps a transport-agnostic *apperr.Error onto the API error
// model, preserving its message. An unknown Kind falls back to ServerError.
func fromAppError(err *apperr.Error) *Error {
	base, ok := appErrorStatus[err.Kind]
	if !ok {
		base = ServerError
	}

	return base.WithMessage(err.Msg)
}

func isHTTPError(err error) bool {
	var he *echo.HTTPError
	return errors.As(err, &he)
}

// fromHTTPError maps framework/middleware *echo.HTTPError (bind failures,
// RequireAuth, route 404/405) onto the api error model so the envelope and
// status stay consistent.
func fromHTTPError(err error) *Error {
	var he *echo.HTTPError

	_ = errors.As(err, &he)

	msg := http.StatusText(he.Code)
	if s, ok := he.Message.(string); ok && s != "" {
		msg = s
	}

	switch he.Code {
	case http.StatusBadRequest:
		return InvalidParams.WithMessage(msg)
	case http.StatusUnauthorized:
		return Unauthorized.WithMessage(msg)
	case http.StatusForbidden:
		return Forbidden.WithMessage(msg)
	case http.StatusNotFound:
		return NotFound.WithMessage(msg)
	case http.StatusConflict:
		return Conflict.WithMessage(msg)
	}

	// Anything else the framework raised in the 4xx range is the CALLER being
	// wrong — 405, 413, 429. Falling through to ServerError told them 500 and
	// paged us for their mistake: an uptime monitor probing with HEAD (405) set
	// off a 5xx alert every five minutes. Keep the framework's status; the
	// envelope carries the generic client-error code.
	if he.Code >= http.StatusBadRequest && he.Code < http.StatusInternalServerError {
		return &Error{Code: InvalidParamsCode, Msg: msg, HTTP: he.Code}
	}

	return ServerError.WithMessage(msg)
}
