// Command remind is the hourly job that sends the daily review reminder through
// the Telegram bot. It is started by cron on the box (see
// infra/learning/cron/install-crons.sh), runs once and exits; it reads only what
// it needs from the environment, so the prod service needs no more than the
// database and the bot.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/log"
	"learning_app/backend/internal/remind"
	"learning_app/backend/internal/tgbot"
)

func main() {
	_ = godotenv.Load()

	log.Setup()

	dsn, token, miniApp := os.Getenv("DATABASE_URL"), os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_MINIAPP_URL")
	if dsn == "" || token == "" || miniApp == "" {
		slog.Error("remind: DATABASE_URL, TELEGRAM_BOT_TOKEN and TELEGRAM_MINIAPP_URL are required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ctx, stop := context.WithTimeout(ctx, 5*time.Minute)
	defer stop()

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		slog.Error("remind: db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	sent, err := remind.Run(ctx, db.New(pool), tgbot.New(token, miniApp), time.Now())
	if err != nil {
		slog.Error("remind: run", "err", err, "sent", sent)
		os.Exit(1)
	}

	slog.Info("remind: done", "sent", sent)
}
