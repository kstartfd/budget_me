package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

func (c Client) anthropicClient(key string) anthropic.Client {
	return anthropic.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(c.urls.Anthropic),
		option.WithHTTPClient(c.http),
		option.WithMaxRetries(0),
	)
}

func (c Client) completeAnthropic(ctx context.Context, req Request) (string, error) {
	client := c.anthropicClient(req.Key)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:        anthropic.Model(req.Model),
		MaxTokens:    16000,
		System:       []anthropic.TextBlockParam{{Text: req.System}},
		OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffort(req.Level)},
		Messages:     []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt))},
	})
	if err != nil {
		return "", anthropicError(err)
	}

	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", errors.New("llm: the model declined the request")
	}

	var text strings.Builder

	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}

	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("llm: the model returned no text")
	}

	return text.String(), nil
}

func (c Client) verifyAnthropic(ctx context.Context, key string) error {
	client := c.anthropicClient(key)

	if _, err := client.Models.List(ctx, anthropic.ModelListParams{}); err != nil {
		return anthropicError(err)
	}

	return nil
}

func anthropicError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return statusError(apiErr.StatusCode)
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", ErrTemporary, err)
	}

	return fmt.Errorf("%w: anthropic request failed", ErrTemporary)
}
