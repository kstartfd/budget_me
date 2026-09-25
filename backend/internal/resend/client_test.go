package resend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendUsesResendContract(t *testing.T) {
	client := New("test-key", "Budget Me <mail@example.com>")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "user-day-slot" {
			t.Errorf("idempotency key = %q", got)
		}
		var body struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Text    string   `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.From != "Budget Me <mail@example.com>" || len(body.To) != 1 || body.To[0] != "user@example.com" || body.Text != "A task" {
			t.Errorf("payload = %+v", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"mail-123"}`))}, nil
	})}
	id, err := client.Send(context.Background(), "user@example.com", "Tasks", "A task", "user-day-slot")
	if err != nil || id != "mail-123" {
		t.Fatalf("send = %q, %v", id, err)
	}
}

func TestSendRejectsProviderFailure(t *testing.T) {
	client := New("key", "mail@example.com")
	client.http = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(`{"message":"private recipient"}`))}, nil
	})}
	if _, err := client.Send(context.Background(), "user@example.com", "Tasks", "body", "key"); err == nil || err.Error() != "Resend returned HTTP 429" {
		t.Fatalf("error = %v", err)
	}
}
