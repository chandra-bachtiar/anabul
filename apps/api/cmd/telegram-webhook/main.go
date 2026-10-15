// cmd/telegram-webhook registers the Telegram webhook for the public ANABUL
// domain and prints its status; takes set or info, needs TELEGRAM_TOKEN, TELEGRAM_WEBHOOK_SECRET, PUBLIC_URL.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"anabul/internal/config"
	"anabul/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "failed:", err)
		os.Exit(1)
	}
}

func run() error {
	command := "set"
	if len(os.Args) > 1 {
		command = strings.TrimSpace(os.Args[1])
	}

	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	baseURL := strings.TrimSpace(os.Getenv("TELEGRAM_API_BASE"))
	client := telegram.NewClient(baseURL, cfg.TelegramToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch command {
	case "set":
		public := strings.TrimSpace(os.Getenv("PUBLIC_URL"))
		if public == "" {
			return fmt.Errorf("PUBLIC_URL is not set, e.g. PUBLIC_URL=https://anabul.example.com")
		}
		webhookURL := strings.TrimRight(public, "/") + "/telegram/webhook"

		if err := client.SetWebhook(ctx, webhookURL, cfg.TelegramWebhookSecret); err != nil {
			return err
		}
		fmt.Printf("webhook registered: %s\n", webhookURL)

		info, err := client.Info(ctx)
		if err == nil {
			printInfo(info)
		}
		return nil

	case "info":
		info, err := client.Info(ctx)
		if err != nil {
			return err
		}
		printInfo(info)
		return nil

	default:
		return fmt.Errorf("unknown command %q, use \"set\" or \"info\"", command)
	}
}

func printInfo(info telegram.WebhookInfo) {
	fmt.Println("webhook status:")
	if info.URL == "" {
		fmt.Println("  url: (not set)")
	} else {
		fmt.Printf("  url: %s\n", info.URL)
	}
	fmt.Printf("  pending updates: %d\n", info.PendingUpdateCount)
	if info.LastErrorMessage != "" {
		fmt.Printf("  last error: %s\n", info.LastErrorMessage)
		fmt.Println("  if the error mentions 401/403, TELEGRAM_WEBHOOK_SECRET must match on the server and here.")
	}
}
