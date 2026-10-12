package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Env                string
	Port               string
	DatabaseURL        string
	CORSOrigins        []string
	AuthSecret         string
	TelegramBotToken   string
	TelegramMiniAppURL string
}

func Load() (Config, error) {
	cfg := Config{
		Env:                value("ENV", "dev"),
		Port:               value("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		CORSOrigins:        strings.Split(value("CORS_ORIGINS", "http://localhost:3001"), ","),
		AuthSecret:         os.Getenv("AUTH_SECRET"),
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramMiniAppURL: value("TELEGRAM_MINIAPP_URL", "http://localhost:3001"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
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
