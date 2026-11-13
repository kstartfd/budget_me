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
	Input           string    `json:"input"`
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
	// Usage counts the cache inside input_tokens, and the thinking inside output_tokens.
	Usage struct {
		InputTokens        int `json:"input_tokens"`
		InputTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c Client) completeOpenAI(ctx context.Context, req Request) (string, Usage, error) {
	var res responsesResponse

	err := c.doJSON(ctx, http.MethodPost, c.urls.OpenAI+"/responses", req.Key, responsesRequest{
		Model:           req.Model,
		Instructions:    req.System,
		Input:           req.Prompt,
		Reasoning:       reasoning{Effort: req.Level},
		MaxOutputTokens: 25000,
	}, &res)
	if err != nil {
		return "", Usage{}, err
	}

	usage := usageOf(res.Usage.InputTokens, res.Usage.InputTokensDetails.CachedTokens, res.Usage.OutputTokens)

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
		return "", usage, errors.New("llm: the model returned no text")
	}

	return text.String(), usage, nil
}
