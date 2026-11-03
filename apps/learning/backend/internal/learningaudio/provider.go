package learningaudio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultGoogleTTSURL = "https://translate.google.com"

type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string { return e.Message }

type GoogleProvider struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewGoogleProvider(baseURL string) GoogleProvider {
	if baseURL == "" {
		baseURL = defaultGoogleTTSURL
	}

	return GoogleProvider{BaseURL: baseURL, HTTPClient: &http.Client{Timeout: 20 * time.Second}}
}

func (p GoogleProvider) Speak(ctx context.Context, text, language string) ([]byte, string, error) {
	endpoint, err := url.Parse(p.BaseURL)
	if err != nil {
		return nil, "", fmt.Errorf("parse tts url: %w", err)
	}

	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/translate_tts"

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	var audio []byte
	for _, chunk := range speechChunks(text) {
		query := endpoint.Query()
		query.Set("ie", "UTF-8")
		query.Set("client", "tw-ob")
		query.Set("tl", NormalizeLanguage(language))
		query.Set("q", chunk)
		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, "", err
		}

		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Accept", "audio/mpeg")

		res, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}

		body, readErr := io.ReadAll(io.LimitReader(res.Body, 16<<20+1))
		closeErr := res.Body.Close()
		if readErr != nil {
			return nil, "", readErr
		}
		if closeErr != nil {
			return nil, "", closeErr
		}

		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			return nil, "", &ProviderError{StatusCode: res.StatusCode, Message: fmt.Sprintf("tts provider returned %d", res.StatusCode)}
		}

		if len(body) == 0 || len(body) > 16<<20 {
			return nil, "", errors.New("tts provider returned an empty or oversized response")
		}

		audio = append(audio, body...)
	}

	return audio, "audio/mpeg", nil
}

func NormalizeLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(code, "_", "-")))
	if i := strings.IndexByte(code, '-'); i > 0 {
		code = code[:i]
	}

	return code
}

func speechChunks(text string) []string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= 180 {
		return []string{text}
	}

	words := strings.Fields(text)
	chunks := make([]string, 0, utf8.RuneCountInString(text)/180+1)
	current := ""

	for _, word := range words {
		if current != "" && utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) > 180 {
			chunks = append(chunks, current)
			current = word
			continue
		}

		if current == "" {
			current = word
		} else {
			current += " " + word
		}
	}

	if current != "" {
		chunks = append(chunks, current)
	}

	return chunks
}
