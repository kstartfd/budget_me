package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/config"
	"learning_app/backend/internal/db"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/log"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/tgbot"
)

func main() {
	_ = godotenv.Load()

	log.Setup()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := db.NewPool(rootCtx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	e := server.New(cfg, pool)
	learningWorker := learningcommands.NewProcessHandler(pool, clock.New())

	workerDone := make(chan struct{})
	go func() { defer close(workerDone); learningWorker.Run(rootCtx) }()

	defer func() { cancel(); <-workerDone }()

	// Greet /start with a "Launch" button (no-op without a bot token).
	go tgbot.New(cfg.TelegramBotToken, cfg.TelegramMiniAppURL).Run(rootCtx)

	go func() {
		<-rootCtx.Done()

		shctx, shcancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shcancel()

		_ = e.Shutdown(shctx)
	}()

	addr := ":" + cfg.Port
	slog.Info("listening", "addr", addr, "env", cfg.Env)

	if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
