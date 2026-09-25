package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://api.resend.com/emails"

// Client sends plain-text notification emails through Resend's HTTP API.
type Client struct {
	key      string
	from     string
	endpoint string
	http     *http.Client
}

func New(key, from string) *Client {
	return &Client{
		key: strings.TrimSpace(key), from: strings.TrimSpace(from),
		endpoint: endpoint, http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Enabled() bool { return c.key != "" && c.from != "" }

// Send returns the provider message ID. The caller supplies a stable key and
// an unchanged body on retries; Resend deduplicates that request for 24 hours.
func (c *Client) Send(ctx context.Context, to, subject, body, idempotencyKey string) (string, error) {
	if !c.Enabled() {
		return "", errors.New("Resend is not configured")
	}

	payload, err := json.Marshal(struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{From: c.from, To: []string{to}, Subject: subject, Text: body})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Resend error bodies may repeat recipient addresses or message content.
		return "", fmt.Errorf("Resend returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", errors.New("Resend returned no email ID")
	}
	return result.ID, nil
}
