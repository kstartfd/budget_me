package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/ratelimit"
)

// defaultDeletionGrace is the wait between "delete my account" and the data
// going; cancelling inside it saves everything.
const defaultDeletionGrace = 30 * 24 * time.Hour

// allTopics is more than a program can have (it holds at most sixteen), for
// the settings that count topics from the first: it means every one.
const allTopics = 32

type Config struct {
	Env         string
	Port        string
	DatabaseURL string
	// AccountDatabaseURL is the shared account database (one account for every app).
	AccountDatabaseURL   string
	AccountDeletionGrace time.Duration
	CORSOrigins          []string
	AuthSecret           string
	TelegramBotToken     string
	TelegramMiniAppURL   string
	AudioCacheDir        string
	// TTSURL is the text-to-speech service root. Never read from the environment:
	// blank means Google Translate's free endpoint, and tests point it at a fake.
	TTSURL string
	// LLMURLs are the provider API roots. Never read from the environment:
	// blank means the real providers, and tests point it at fakes.
	LLMURLs llm.URLs
	// OCRBin is the tesseract program that reads pictures. Never read from the
	// environment: blank means the one on PATH, and tests point it at a stand-in.
	OCRBin string
	// ChatRate is how fast one user may use the chat. Never read from the
	// environment: blank means the defaults, and tests loosen or tighten it.
	ChatRate ratelimit.Config
	// PathPrefetch is how many topics of a new program, from the first, get
	// their exercises written in the background right after it is built. Never
	// read from the environment: Load sets the default (every topic, as a program
	// has at most sixteen), tests leave it at zero so the stub provider only sees
	// the calls a test makes itself.
	PathPrefetch int
}

func Load() (Config, error) {
	cfg := Config{
		Env:                  value("ENV", "dev"),
		Port:                 value("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		AccountDatabaseURL:   os.Getenv("ACCOUNT_DATABASE_URL"),
		AccountDeletionGrace: durationOrDefault("ACCOUNT_DELETION_GRACE", defaultDeletionGrace),
		CORSOrigins:          strings.Split(value("CORS_ORIGINS", "http://localhost:3001"), ","),
		AuthSecret:           os.Getenv("AUTH_SECRET"),
		TelegramBotToken:     os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramMiniAppURL:   value("TELEGRAM_MINIAPP_URL", "http://localhost:3001"),
		AudioCacheDir:        value("AUDIO_CACHE_DIR", "/var/lib/learning/audio"),
		PathPrefetch:         allTopics,
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if cfg.AccountDatabaseURL == "" {
		return cfg, errors.New("ACCOUNT_DATABASE_URL is required")
	}
	if cfg.AuthSecret == "" {
		return cfg, errors.New("AUTH_SECRET is required")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return fallback
}
