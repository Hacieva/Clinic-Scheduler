package main

import (
	"log/slog"
	"os"
	"time"
)

func main() {
	slog.Info("bot starting...")

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		slog.Warn("BOT_TOKEN not set, running in stub mode")
	}

	for {
		time.Sleep(30 * time.Second)
	}
}
