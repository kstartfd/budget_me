// Package ai is the advice writer: it turns numbers the backend already
// computed into a sentence a human wants to read. It never decides anything —
// no amounts, no comparisons, no limits come from the model, only wording.
//
// It is deliberately kept out of the request path. Measured on the configured
// model (glm-4.5-flash via z.ai): 3–18 seconds per call, and with reasoning left
// on the answer came back EMPTY — the token budget went entirely into the
// model's internal monologue (finish_reason "length"). So: reasoning disabled,
// hard timeout, and callers run it in the background and cache the result.
package ai

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

// callTimeout caps one generation. Well past the ~3s the model usually needs,
// short enough that a hung provider cannot pin a goroutine for minutes.
const callTimeout = 30 * time.Second

var ErrDisabled = errors.New("ai: no API key configured")

type Client struct {
	key     string
	model   string
	baseURL string
	http    *http.Client
}

func New(key, model, baseURL string) *Client {
	return &Client{
		key:     key,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: callTimeout},
	}
}

// Enabled reports whether a key is configured. Without one the app still works:
// callers show the facts and skip the prose.
func (c *Client) Enabled() bool { return c != nil && c.key != "" }

// Write asks the model to phrase `prompt` and returns its plain text.
func (c *Client) Write(ctx context.Context, system, prompt string) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}

	body, err := json.Marshal(chatRequest{
		Model: c.model,
		// Reasoning off: the model is a thinking one, and left on it spends the
		// whole token budget on its monologue and returns an empty message.
		Thinking:  thinking{Type: "disabled"},
		MaxTokens: 400,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: provider status %d", res.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("ai: empty response")
	}

	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		return "", errors.New("ai: model returned no text")
	}

	return text, nil
}

type chatRequest struct {
	Model     string    `json:"model"`
	Thinking  thinking  `json:"thinking"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type thinking struct {
	Type string `json:"type"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
