package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	bot, err := NewBot(cfg, games)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := bot.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}
