package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type chatRequest struct {
	Model           string   `json:"model"`
	Messages        []any    `json:"messages"`
	Thinking        thinking `json:"thinking"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
	MaxTokens       int      `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// visionMessage is a user message made of parts, which is how a picture
// reaches the model; a plain message keeps its content as one string.
type visionMessage struct {
	Role    string       `json:"role"`
	Content []visionPart `json:"content"`
}

type visionPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *visionImageURL `json:"image_url,omitempty"`
}

type visionImageURL struct {
	URL string `json:"url"`
}

func userMessage(req Request) any {
	if len(req.Images) == 0 {
		return chatMessage{Role: "user", Content: req.Prompt}
	}

	parts := make([]visionPart, 0, len(req.Images)+1)
	for _, image := range req.Images {
		parts = append(parts, visionPart{Type: "image_url", ImageURL: &visionImageURL{URL: image.dataURL()}})
	}

	return visionMessage{Role: "user", Content: append(parts, visionPart{Type: "text", Text: req.Prompt})}
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
		Messages: []any{
			chatMessage{Role: "system", Content: req.System},
			userMessage(req),
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
