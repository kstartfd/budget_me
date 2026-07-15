// Package httpx is the HTTP plumbing shared by every feature: the response
// envelope, the API error model, and the cross-cutting middleware. Features
// import httpx; httpx imports no feature. Mapping a domain sentinel onto an
// HTTP status therefore happens at each feature's handler boundary, not here.
package httpx

import (
	"fmt"
	"net/http"
)

// Error is a standardized API error: a stable business Code, an HTTP status, a
// message and optional details.
type Error struct {
	Code    int      `json:"code"`
	Msg     string   `json:"message"`
	Details []string `json:"details,omitempty"`
	HTTP    int      `json:"-"`
}

var codes = map[int]bool{}

// Business codes. Stable across releases — the frontend switches on these.
const (
	SuccessCode       = 0
	ServerErrorCode   = 10000
	InvalidParamsCode = 10001
	NotFoundCode      = 10002
	UnauthorizedCode  = 20001
	ForbiddenCode     = 20002
	ConflictCode      = 40001
)

// Predefined errors. New panics on a duplicate code (registry guard).
var (
	ServerError   = New(ServerErrorCode, http.StatusInternalServerError, "internal error")
	InvalidParams = New(InvalidParamsCode, http.StatusBadRequest, "invalid params")
	NotFound      = New(NotFoundCode, http.StatusNotFound, "not found")
	Unauthorized  = New(UnauthorizedCode, http.StatusUnauthorized, "unauthorized")
	Forbidden     = New(ForbiddenCode, http.StatusForbidden, "forbidden")
	Conflict      = New(ConflictCode, http.StatusConflict, "conflict")
)

// New registers a predefined error and panics on a duplicate code.
func New(code, httpStatus int, msg string) *Error {
	if codes[code] {
		panic(fmt.Sprintf("httpx: duplicate error code %d", code))
	}

	codes[code] = true

	return &Error{Code: code, HTTP: httpStatus, Msg: msg}
}

func (e Error) Error() string { return e.Msg }

// StatusCode returns the HTTP status for this error.
func (e *Error) StatusCode() int { return e.HTTP }

// WithMessage returns a copy carrying a custom message (same code/status).
func (e *Error) WithMessage(msg string) *Error {
	c := *e
	c.Msg = msg

	return &c
}
