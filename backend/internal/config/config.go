package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

// defaultDeletionGrace — 30 days between "delete my account" and the row
// actually going away.
const defaultDeletionGrace = 30 * 24 * time.Hour

// Config holds the process configuration, loaded once from the environment.
type Config struct {
	Env         string
	Port        string
	DatabaseURL string

	CORSOrigins []string

	// AuthSecret signs session tokens (util/token). Required — a guessable
	// secret means forgeable sessions.
	AuthSecret string
	// TelegramBotToken verifies Mini App initData. Optional: without it the
	// web login/registration still works, Telegram auto-login returns 401.
	TelegramBotToken string
	// TelegramMiniAppURL is the HTTPS URL the bot's /start "Launch" button
	// opens. Also gates the /start long-poll greeter (tgbot).
	TelegramMiniAppURL string

	// NBRBBaseURL and BelarusbankBaseURL are the exchange-rate API bases used
	// for the "≈$" conversion (National Bank of Belarus + Belarusbank; both
	// keyless public feeds).
	NBRBBaseURL        string
	BelarusbankBaseURL string

	// AI* configure the advice writer (see internal/ai). The model only phrases
	// numbers the backend already computed, and it is NEVER called inside a
	// request — it takes seconds to answer, so generation happens in the
	// background and the API serves a cached text. An empty key simply disables
	// the writer: the app then shows the facts without the prose.
	AIAPIKey  string
	AIModel   string
	AIBaseURL string

	// AlertChatID is the Telegram chat operational alerts go to (5xx from the
	// API; the on-box cron sends its own). Empty disables alerting entirely —
	// dev boxes stay quiet. Falls back to BACKUP_CHAT_ID so a single chat can
	// serve both the nightly dumps and the alerts until the noise justifies two.
	AlertChatID string

	// AccountDeletionGrace is how long a requested account deletion waits
	// before cmd/worker actually removes it — the window in which signing back
	// in and cancelling still saves the account. Both the API (which reports
	// the deadline) and the worker (which enforces it) read this one value.
	AccountDeletionGrace time.Duration

	// UploadsDir is where item photos are saved (see internal/photostore) and
	// served from (public.Static in server.go). Relative in dev — the backend
	// container's CWD is the bind-mounted backend/ — absolute in prod, on a
	// named volume so photos survive redeploys.
	UploadsDir string

	// Empty credentials disable Day email notifications without affecting the
	// API or the existing worker jobs.
	ResendAPIKey    string
	ResendFromEmail string
}

// Load reads the configuration from the environment, returning an error when a
// required value (DATABASE_URL, AUTH_SECRET) is missing.
func Load() (Config, error) {
	cfg := Config{
		Env:                orDefault("ENV", "dev"),
		Port:               orDefault("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		CORSOrigins:        splitCSV(orDefault("CORS_ORIGINS", "http://localhost:3000")),
		AuthSecret:         os.Getenv("AUTH_SECRET"),
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramMiniAppURL: orDefault("TELEGRAM_MINIAPP_URL", "https://budgetme.xyz"),

		NBRBBaseURL:        orDefault("NBRB_BASE_URL", "https://api.nbrb.by"),
		BelarusbankBaseURL: orDefault("BELARUSBANK_BASE_URL", "https://belarusbank.by"),

		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIModel:   orDefault("AI_MODEL", "glm-4.5-flash"),
		AIBaseURL: orDefault("AI_BASE_URL", "https://api.z.ai/api/paas/v4"),

		AlertChatID: orDefault("ALERT_CHAT_ID", os.Getenv("BACKUP_CHAT_ID")),

		AccountDeletionGrace: durationOrDefault(
			"ACCOUNT_DELETION_GRACE",
			defaultDeletionGrace,
		),

		UploadsDir:      orDefault("UPLOADS_DIR", "data/uploads"),
		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFromEmail: os.Getenv("RESEND_FROM_EMAIL"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}

	if cfg.AuthSecret == "" {
		return cfg, errors.New("AUTH_SECRET is required")
	}

	return cfg, nil
}

func orDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// durationOrDefault parses a Go duration ("720h", "15m"). An unparsable value
// falls back rather than failing the boot: a typo here must not take the API
// down, and the fallback is the policy we ship with anyway.
func durationOrDefault(key string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(os.Getenv(key))
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}
