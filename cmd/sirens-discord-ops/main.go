package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coilysiren/sirens-discord-ops/internal/bot"
)

func main() {
	cfg, err := bot.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	b, err := bot.NewBot(cfg, bot.Games)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := b.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}
