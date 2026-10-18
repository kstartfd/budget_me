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

	"budget_me/backend/internal/config"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/log"
	"budget_me/backend/internal/server"
	"budget_me/backend/internal/tgbot"
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

	accountPool, err := db.NewPool(rootCtx, cfg.AccountDatabaseURL)
	if err != nil {
		slog.Error("account db connect", "err", err)
		os.Exit(1)
	}
	defer accountPool.Close()

	e := server.New(cfg, pool, accountPool)

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
