package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type responsesRequest struct {
	Model           string    `json:"model"`
	Instructions    string    `json:"instructions"`
	Input           any       `json:"input"`
	Reasoning       reasoning `json:"reasoning"`
	MaxOutputTokens int       `json:"max_output_tokens"`
}

type reasoning struct {
	Effort string `json:"effort"`
}

type responsesResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

// responsesInput is the prompt alone, or, with pictures, one user message that
// holds the pictures and then the prompt.
func responsesInput(req Request) any {
	if len(req.Images) == 0 {
		return req.Prompt
	}

	parts := make([]map[string]string, 0, len(req.Images)+1)
	for _, image := range req.Images {
		parts = append(parts, map[string]string{"type": "input_image", "image_url": image.dataURL()})
	}

	parts = append(parts, map[string]string{"type": "input_text", "text": req.Prompt})

	return []map[string]any{{"role": "user", "content": parts}}
}

func (c Client) completeOpenAI(ctx context.Context, req Request) (string, error) {
	var res responsesResponse

	err := c.doJSON(ctx, http.MethodPost, c.urls.OpenAI+"/responses", req.Key, responsesRequest{
		Model:           req.Model,
		Instructions:    req.System,
		Input:           responsesInput(req),
		Reasoning:       reasoning{Effort: req.Level},
		MaxOutputTokens: 25000,
	}, &res)
	if err != nil {
		return "", err
	}

	var text strings.Builder

	for _, item := range res.Output {
		if item.Type != "message" {
			continue
		}

		for _, part := range item.Content {
			if part.Type == "output_text" {
				text.WriteString(part.Text)
			}
		}
	}

	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("llm: the model returned no text")
	}

	return text.String(), nil
}
