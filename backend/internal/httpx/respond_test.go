package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"budget_me/backend/internal/apperr"
)

type recorder struct {
	method string
	route  string
	status int
	calls  int
}

func (r *recorder) Report(method, route string, status int, _ error) {
	r.calls++
	r.method, r.route, r.status = method, route, status
}

// The error sink is the ONE place a 5xx passes through — if this wiring breaks,
// alerting goes silent everywhere at once and nothing else notices.
func TestErrorHandlerReportsServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCalls  int
		wantStatus int
	}{
		{
			name:       "unexpected failure is a 500 and pages",
			err:        errors.New("connection refused"),
			wantCalls:  1,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "invalid input is the caller being wrong — no alert",
			err:       apperr.NewInvalid("amount must be positive"),
			wantCalls: 0,
		},
		{
			name:      "not found is not an incident",
			err:       apperr.NewNotFound("plan not found"),
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			e := echo.New()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/abc-123", nil)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			// Echo fills this from the matched route; set it by hand here.
			c.SetPath("/api/v1/transactions/:id")

			NewErrorHandler(rec)(tt.err, c)

			if rec.calls != tt.wantCalls {
				t.Fatalf("репортер вызван %d раз, ждали %d", rec.calls, tt.wantCalls)
			}

			if tt.wantCalls == 0 {
				return
			}

			if rec.status != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.status, tt.wantStatus)
			}

			// The ROUTE, not the URL: a real transaction id must never reach the chat.
			if rec.route != "/api/v1/transactions/:id" || rec.method != http.MethodPost {
				t.Errorf("репортер получил %s %s — ждали шаблон маршрута", rec.method, rec.route)
			}
		})
	}
}

// The framework's own 4xx must stay 4xx. They used to fall through to
// ServerError: an uptime monitor probing /healthz with HEAD got a 405 rendered
// as 500, and paged the owner every five minutes for a service that was fine.
func TestFrameworkClientErrorsStayClientErrors(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{name: "method not allowed", code: http.StatusMethodNotAllowed},
		{name: "too many requests", code: http.StatusTooManyRequests},
		{name: "payload too large", code: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			e := echo.New()
			res := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodHead, "/healthz", nil), res)

			NewErrorHandler(rec)(echo.NewHTTPError(tt.code), c)

			if res.Code != tt.code {
				t.Errorf("status = %d, want %d", res.Code, tt.code)
			}

			if rec.calls != 0 {
				t.Errorf("клиентская ошибка подняла %d алерт(ов)", rec.calls)
			}
		})
	}
}

// A 5xx from the framework is still ours to answer for.
func TestFrameworkServerErrorsStillPage(t *testing.T) {
	rec := &recorder{}
	e := echo.New()
	res := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/me", nil), res)

	NewErrorHandler(rec)(echo.NewHTTPError(http.StatusBadGateway), c)

	if rec.calls != 1 {
		t.Fatalf("502 должен алертить, вызовов: %d", rec.calls)
	}
}

// A nil reporter is the dev/no-chat case: the handler must still answer.
func TestErrorHandlerWithoutReporter(t *testing.T) {
	e := echo.New()
	res := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/me", nil), res)

	NewErrorHandler(nil)(errors.New("boom"), c)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
}
