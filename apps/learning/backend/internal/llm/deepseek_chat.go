package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxStreamLine bounds one server-sent line; a reply chunk is a few bytes, so
// this only stops a broken stream from filling memory.
const maxStreamLine = 1 << 20

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatStreamRequest struct {
	Model         string        `json:"model"`
	Messages      []chatMessage `json:"messages"`
	Thinking      thinking      `json:"thinking"`
	MaxTokens     int           `json:"max_tokens"`
	Stream        bool          `json:"stream"`
	StreamOptions streamOptions `json:"stream_options"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens         int `json:"prompt_tokens"`
		CompletionTokens     int `json:"completion_tokens"`
		PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
	} `json:"usage"`
}

// chatDeepSeek answers a tutoring question, which needs no reasoning: thinking
// stays off, so the first words arrive at once and nothing is paid for hidden
// thoughts. Only `content` is read, so a reasoning trace could never reach the
// stored history.
func (c Client) chatDeepSeek(ctx context.Context, req ChatRequest, profile ChatProfile, onDelta func(string)) (ChatResult, error) {
	messages := make([]chatMessage, 0, len(req.Messages)+1)
	messages = append(messages, chatMessage{Role: "system", Content: req.System})

	for _, m := range req.Messages {
		messages = append(messages, chatMessage{Role: m.Role, Content: m.Content})
	}

	raw, err := json.Marshal(chatStreamRequest{
		Model:         req.Model,
		Messages:      messages,
		Thinking:      thinking{Type: "disabled"},
		MaxTokens:     profile.MaxReply,
		Stream:        true,
		StreamOptions: streamOptions{IncludeUsage: true},
	})
	if err != nil {
		return ChatResult{}, err
	}

	url := c.urls.DeepSeek + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return ChatResult{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.Key)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return ChatResult{}, fmt.Errorf("%w: %s", ErrTemporary, redact(err, url))
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return ChatResult{}, statusError(res.StatusCode)
	}

	return readChatStream(res.Body, onDelta)
}

// readChatStream follows an OpenAI-style event stream: "data: {json}" lines,
// ending with "data: [DONE]". Comment lines (": keep-alive") are ignored.
func readChatStream(body io.Reader, onDelta func(string)) (ChatResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxStreamLine)

	var (
		text   strings.Builder
		result ChatResult
		done   bool
	)

	for scanner.Scan() {
		payload, ok := strings.CutPrefix(scanner.Text(), "data:")
		if !ok {
			continue
		}

		payload = strings.TrimSpace(payload)
		if payload == "[DONE]" {
			done = true

			break
		}

		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return ChatResult{}, fmt.Errorf("llm: unreadable provider stream: %w", err)
		}

		for _, choice := range chunk.Choices {
			if piece := choice.Delta.Content; piece != "" {
				text.WriteString(piece)

				if onDelta != nil {
					onDelta(piece)
				}
			}
		}

		if chunk.Usage != nil {
			result.PromptTokens = chunk.Usage.PromptTokens
			result.CompletionTokens = chunk.Usage.CompletionTokens
			result.CacheHitTokens = chunk.Usage.PromptCacheHitTokens
		}
	}

	if scanner.Err() != nil {
		return ChatResult{}, fmt.Errorf("%w: the provider stream broke", ErrTemporary)
	}

	if !done {
		return ChatResult{}, fmt.Errorf("%w: the provider stream ended early", ErrTemporary)
	}

	if strings.TrimSpace(text.String()) == "" {
		return ChatResult{}, errors.New("llm: the model returned no text")
	}

	result.Text = text.String()

	return result, nil
}
