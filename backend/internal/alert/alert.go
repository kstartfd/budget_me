// Package alert delivers operational alerts to Telegram through the bot the app
// already owns (@money_carrot_bot) — no Prometheus, no Grafana, nothing resident
// on a 1 GB box where the monitoring would compete for the very memory it is
// meant to watch.
//
// It is the in-process half of the alerting. The other half is on-box cron
// (infra/scripts/healthcheck.sh: disk, memory, load, containers, backup
// freshness) and an EXTERNAL uptime check — an alert that runs on the box cannot
// fire when the box is what died.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// sendTimeout caps a delivery: an alert that hangs must not hold anything up.
const sendTimeout = 15 * time.Second

// Client sends messages as the bot. Zero value / empty config is DISABLED —
// alerting is optional infrastructure, and a missing chat id must never turn
// into a failed request or a panic in a live handler.
type Client struct {
	botToken string
	chatID   string
	baseURL  string
	http     *http.Client
}

// New builds a client. Either field empty → Enabled() is false and every Send is
// a no-op, which is exactly what a dev box (and a prod box before the operator
// sets ALERT_CHAT_ID) should do.
func New(botToken, chatID string) *Client {
	return &Client{
		botToken: botToken,
		chatID:   chatID,
		baseURL:  "https://api.telegram.org",
		http:     &http.Client{Timeout: sendTimeout},
	}
}

// Enabled reports whether alerts can actually be delivered.
func (c *Client) Enabled() bool {
	return c != nil && c.botToken != "" && c.chatID != ""
}

// Send delivers one message. Errors are logged, never returned: an alert is a
// side channel, and nothing in the app may fail because the side channel did.
func (c *Client) Send(ctx context.Context, text string) {
	if !c.Enabled() {
		return
	}

	body, err := json.Marshal(map[string]string{
		"chat_id": c.chatID,
		"text":    text,
	})
	if err != nil {
		slog.Warn("alert: marshal failed", "error", err)

		return
	}

	endpoint := fmt.Sprintf(
		"%s/bot%s/sendMessage", c.baseURL, url.PathEscape(c.botToken),
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewReader(body),
	)
	if err != nil {
		slog.Warn("alert: request failed", "error", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		slog.Warn("alert: send failed", "error", err)

		return
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		slog.Warn("alert: telegram rejected the alert", "status", res.StatusCode)
	}
}
