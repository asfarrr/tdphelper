package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"

	"tdphelper/internal/config"
	"tdphelper/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	b, err := bot.New(cfg.BotToken)
	if err != nil {
		slog.Error("create bot", "err", err)
		os.Exit(1)
	}
	handlers.Register(b)

	slog.Info("bot started (long polling)")
	b.Start(ctx)
}
