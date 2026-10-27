// Package llm calls the three providers a user can pick an agent from. It
// only phrases text: every call is one system prompt and one prompt in, plain
// text out, with the user's own API key.
package llm

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	// ErrTemporary marks failures worth retrying later: timeouts, rate
	// limits, provider outages.
	ErrTemporary = errors.New("llm: temporary failure")
	// ErrRejected means the provider does not accept the API key.
	ErrRejected = errors.New("llm: api key rejected")
)

type URLs struct {
	Anthropic string
	OpenAI    string
	DeepSeek  string
}

var DefaultURLs = URLs{
	Anthropic: "https://api.anthropic.com",
	OpenAI:    "https://api.openai.com/v1",
	DeepSeek:  "https://api.deepseek.com",
}

type Request struct {
	Provider Provider
	Model    string
	Level    string
	Key      string
	System   string
	Prompt   string
}

type Client struct {
	urls URLs
	http *http.Client
}

// New fills any blank root with the real provider's and leaves the timeout to
// the caller's context: how long a call may take depends on the level the user
// picked.
func New(urls URLs) Client {
	urls.Anthropic = cmp.Or(urls.Anthropic, DefaultURLs.Anthropic)
	urls.OpenAI = cmp.Or(urls.OpenAI, DefaultURLs.OpenAI)
	urls.DeepSeek = cmp.Or(urls.DeepSeek, DefaultURLs.DeepSeek)

	return Client{urls: urls, http: &http.Client{}}
}

func (c Client) Complete(ctx context.Context, req Request) (string, error) {
	switch req.Provider {
	case Anthropic:
		return c.completeAnthropic(ctx, req)
	case OpenAI:
		return c.completeOpenAI(ctx, req)
	case DeepSeek:
		return c.completeDeepSeek(ctx, req)
	}

	return "", fmt.Errorf("llm: unknown provider %q", req.Provider)
}

// Verify asks the provider for its model list, which costs nothing. Only an
// explicit "invalid key" answer is an error worth reporting; any other failure
// says nothing about the key.
func (c Client) Verify(ctx context.Context, provider Provider, key string) error {
	switch provider {
	case Anthropic:
		return c.verifyAnthropic(ctx, key)
	case OpenAI:
		return c.verifyBearer(ctx, c.urls.OpenAI+"/models", key)
	case DeepSeek:
		return c.verifyBearer(ctx, c.urls.DeepSeek+"/models", key)
	}

	return fmt.Errorf("llm: unknown provider %q", provider)
}

func (c Client) verifyBearer(ctx context.Context, url, key string) error {
	return c.doJSON(ctx, http.MethodGet, url, key, nil, nil)
}

// statusError sorts a provider's HTTP status into the errors callers branch on.
func statusError(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return ErrRejected
	case status == http.StatusRequestTimeout, status == http.StatusConflict, status == http.StatusTooManyRequests, status >= 500:
		return fmt.Errorf("%w: provider status %d", ErrTemporary, status)
	}

	return fmt.Errorf("llm: provider status %d", status)
}

// doJSON sends one bearer-authenticated request. Its errors never carry the
// key or the response body, so they are safe to log.
func (c Client) doJSON(ctx context.Context, method, url, key string, in, out any) error {
	var body io.Reader

	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}

		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrTemporary, redact(err, url))
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return statusError(res.StatusCode)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("llm: unreadable provider response: %w", err)
	}

	return nil
}

// redact drops the URL net/http puts in transport errors; the host is all a
// log line needs.
func redact(err error, url string) string {
	return strings.ReplaceAll(err.Error(), url, "provider")
}
