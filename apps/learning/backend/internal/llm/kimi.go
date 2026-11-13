package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// kimiRequest is Moonshot's chat completions call for the K3 models: the same
// shape as DeepSeek's, but thinking is always on, so the level only says how
// hard it reasons, and the cap on the answer is max_completion_tokens (the
// fixed sampling settings are left out, the API rejects other values).
type kimiRequest struct {
	Model               string        `json:"model"`
	Messages            []chatMessage `json:"messages"`
	ReasoningEffort     string        `json:"reasoning_effort"`
	MaxCompletionTokens int           `json:"max_completion_tokens"`
}

// completeKimi reads only `content` of the answer, so the reasoning trace the
// API returns next to it (reasoning_content) is never kept.
func (c Client) completeKimi(ctx context.Context, req Request) (string, Usage, error) {
	var res chatResponse

	err := c.doJSON(ctx, http.MethodPost, c.urls.Kimi+"/chat/completions", req.Key, kimiRequest{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		ReasoningEffort:     req.Level,
		MaxCompletionTokens: 32000,
	}, &res)
	if err != nil {
		return "", Usage{}, err
	}

	if len(res.Choices) == 0 || strings.TrimSpace(res.Choices[0].Message.Content) == "" {
		return "", res.usage(), errors.New("llm: the model returned no text")
	}

	return res.Choices[0].Message.Content, res.usage(), nil
}
