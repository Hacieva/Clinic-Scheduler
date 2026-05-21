package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mymmrac/telego"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
	"github.com/Hacieva/clinic-scheduler/bot/internal/flow"
	"github.com/Hacieva/clinic-scheduler/bot/internal/handler"
	"github.com/Hacieva/clinic-scheduler/bot/internal/session"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		slog.Error("BOT_TOKEN is not set")
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		slog.Error("BACKEND_URL is not set")
		os.Exit(1)
	}

	botSecret := os.Getenv("BOT_API_SECRET")
	if botSecret == "" {
		slog.Error("BOT_API_SECRET is not set")
		os.Exit(1)
	}

	// Graceful shutdown: cancel context on SIGINT/SIGTERM.
	// This closes the updates channel and stops the polling loop cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// — Compose dependencies (no package-level globals) —
	sessions := session.NewPostgresStore(pool)
	apiClient := client.New(backendURL, botSecret)

	// WithDiscardLogger silences telego's internal logger; bot uses slog only.
	bot, err := telego.NewBot(token, telego.WithDiscardLogger())
	if err != nil {
		slog.Error("failed to create bot", "err", err)
		os.Exit(1)
	}

	sender := handler.NewTelegramSender(bot)
	fsm := flow.NewHandler(sessions, apiClient, sender)
	dispatcher := handler.NewDispatcher(fsm)

	// UpdatesViaLongPolling closes the channel when ctx is cancelled.
	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		slog.Error("failed to start long polling", "err", err)
		os.Exit(1)
	}

	slog.Info("bot started")

	for update := range updates {
		// Each update is handled in its own goroutine.
		// Dispatcher.Handle has a deferred recover — panics are logged, not fatal.
		go dispatcher.Handle(ctx, update)
	}

	slog.Info("bot stopped")
}
