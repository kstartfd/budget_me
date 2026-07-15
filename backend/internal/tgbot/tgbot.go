// Package tgbot runs a Telegram long-polling worker that greets users on
// /start with a message and a button that launches the Mini App. Long polling
// needs no public URL, so it works from localhost as well as production.
package tgbot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.telegram.org"

// Worker long-polls the Telegram Bot API and replies to /start.
type Worker struct {
	token      string
	miniAppURL string
	client     *http.Client
}

// New returns a Worker, or nil (a no-op) when the bot token is empty.
func New(token, miniAppURL string) *Worker {
	if token == "" {
		return nil
	}
	// Client timeout must exceed the getUpdates long-poll timeout below.
	return &Worker{
		token:      token,
		miniAppURL: miniAppURL,
		client:     &http.Client{Timeout: 65 * time.Second},
	}
}

// Run polls getUpdates until ctx is cancelled, greeting every /start. Safe to
// call on a nil Worker (returns immediately).
func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	// getUpdates and a webhook are mutually exclusive; clear any webhook first.
	w.dropWebhook(ctx)
	slog.Info("tgbot: polling for /start")

	var offset int64

	for {
		if ctx.Err() != nil {
			return
		}

		updates, err := w.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.Warn("tgbot: getUpdates", "err", err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}

			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1
			w.handle(ctx, u)
		}
	}
}

type update struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

func (w *Worker) getUpdates(ctx context.Context, offset int64) ([]update, error) {
	q := url.Values{}
	q.Set("offset", strconv.FormatInt(offset, 10))
	q.Set("timeout", "50")
	q.Set("allowed_updates", `["message"]`)

	var out struct {
		OK     bool     `json:"ok"`
		Result []update `json:"result"`
	}

	if err := w.call(ctx, "getUpdates", q, nil, &out); err != nil {
		return nil, err
	}

	return out.Result, nil
}

// handle replies to a /start message (with or without a deep-link payload).
func (w *Worker) handle(ctx context.Context, u update) {
	if u.Message == nil {
		return
	}

	text := strings.TrimSpace(u.Message.Text)
	if text != "/start" && !strings.HasPrefix(text, "/start ") {
		return
	}

	w.sendStart(ctx, u.Message.Chat.ID)
}

func (w *Worker) sendStart(ctx context.Context, chatID int64) {
	body := map[string]any{
		"chat_id": chatID,
		"text":    "Welcome! Open budget_me 👋",
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]any{{
				{"text": "🚀 Launch", "web_app": map[string]string{"url": w.miniAppURL}},
			}},
		},
	}
	if err := w.call(ctx, "sendMessage", nil, body, nil); err != nil {
		slog.Warn("tgbot: sendMessage", "err", err)
	}
}

func (w *Worker) dropWebhook(ctx context.Context) {
	_ = w.call(ctx, "deleteWebhook", nil, nil, nil)
}

// call invokes a Bot API method: GET with query when body is nil, else POST
// JSON. Decodes into out when non-nil. The token-bearing URL is never logged.
func (w *Worker) call(ctx context.Context, method string, q url.Values, body, out any) error {
	endpoint := apiBase + "/bot" + w.token + "/" + method

	var req *http.Request

	var err error

	if body != nil {
		buf, mErr := json.Marshal(body)
		if mErr != nil {
			return mErr
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		if len(q) > 0 {
			endpoint += "?" + q.Encode()
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	}

	if err != nil {
		return err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}
