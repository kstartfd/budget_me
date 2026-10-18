package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

// defaultDeletionGrace is the wait between "delete my account" and the data
// going; cancelling inside it saves everything.
const defaultDeletionGrace = 30 * 24 * time.Hour

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
