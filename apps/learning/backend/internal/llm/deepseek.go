package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type chatRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	Thinking        thinking      `json:"thinking"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
	MaxTokens       int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type thinking struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// completeDeepSeek maps the picker's "off" onto the thinking switch; every
// other level turns thinking on at that effort.
func (c Client) completeDeepSeek(ctx context.Context, req Request) (string, error) {
	body := chatRequest{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		Thinking:  thinking{Type: "enabled"},
		MaxTokens: 32000,
	}

	if req.Level == "off" {
		body.Thinking = thinking{Type: "disabled"}
	} else {
		body.ReasoningEffort = req.Level
	}

	var res chatResponse
	if err := c.doJSON(ctx, http.MethodPost, c.urls.DeepSeek+"/chat/completions", req.Key, body, &res); err != nil {
		return "", err
	}

	if len(res.Choices) == 0 || strings.TrimSpace(res.Choices[0].Message.Content) == "" {
		return "", errors.New("llm: the model returned no text")
	}

	return res.Choices[0].Message.Content, nil
}
