package learningaudio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizeLanguage(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"en-US", "en"},
		{"  pt_BR ", "pt"},
		{"zh-CN", "zh"},
		{"RU", "ru"},
	} {
		if got := NormalizeLanguage(tc.in); got != tc.want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSpeechChunksStayWithinProviderLimit(t *testing.T) {
	word := strings.Repeat("длинношеее", 12)
	text := strings.TrimSuffix(strings.Repeat(word+" ", 20), " ")

	chunks := speechChunks("  " + text + "  ")
	if len(chunks) < 2 {
		t.Fatalf("expected several chunks, got %d", len(chunks))
	}

	joined := strings.Join(chunks, " ")
	if joined != strings.Join(strings.Fields(text), " ") {
		t.Fatal("chunks did not preserve the text")
	}

	for _, chunk := range chunks {
		if utf8.RuneCountInString(chunk) > 180 {
			t.Fatalf("chunk is too long: %d", utf8.RuneCountInString(chunk))
		}
	}
}

func TestGoogleProviderRequestsNormalizedChunks(t *testing.T) {
	var queries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate_tts" {
			t.Errorf("path = %q", r.URL.Path)
		}

		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("mp3:" + r.URL.Query().Get("q") + ";"))
	}))
	defer srv.Close()

	provider := NewGoogleProvider(srv.URL)
	audio, contentType, err := provider.Speak(context.Background(), strings.Repeat("say this ", 40), "EN-us")
	if err != nil {
		t.Fatal(err)
	}

	if contentType != "audio/mpeg" {
		t.Fatalf("content type = %q", contentType)
	}
	if len(queries) < 2 {
		t.Fatalf("one request for a long phrase: %v", queries)
	}
	if !strings.HasPrefix(string(audio), "mp3:") || !strings.Contains(string(audio), "say this") {
		t.Fatalf("audio = %q", audio)
	}

	for _, rawQuery := range queries {
		if !strings.Contains(rawQuery, "tl=en") || !strings.Contains(rawQuery, "client=tw-ob") {
			t.Fatalf("query lost language or client: %q", rawQuery)
		}
	}
}

func TestGoogleProviderReportsProviderFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, err := NewGoogleProvider(srv.URL).Speak(context.Background(), "hello", "en")
	if err == nil {
		t.Fatal("expected an error")
	}

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusNotFound {
		t.Fatalf("error = %T %v", err, err)
	}
}
