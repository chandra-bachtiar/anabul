// cmd/telegram-webhook mendaftarkan webhook Telegram ke domain publik ANABUL
// dan menampilkan statusnya. Butuh TELEGRAM_TOKEN, TELEGRAM_WEBHOOK_SECRET,
// dan PUBLIC_URL (mis. https://anabul.contoh.com) di environment atau .env.
//
// Contoh:
//
//	go run ./cmd/telegram-webhook set
//	go run ./cmd/telegram-webhook info
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
	if err := jalankan(); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func jalankan() error {
	perintah := "set"
	if len(os.Args) > 1 {
		perintah = strings.TrimSpace(os.Args[1])
	}

	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("baca .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	baseURL := strings.TrimSpace(os.Getenv("TELEGRAM_API_BASE"))
	client := telegram.NewClient(baseURL, cfg.TelegramToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch perintah {
	case "set":
		public := strings.TrimSpace(os.Getenv("PUBLIC_URL"))
		if public == "" {
			return fmt.Errorf("PUBLIC_URL belum diisi, contoh: PUBLIC_URL=https://anabul.contoh.com")
		}
		webhookURL := strings.TrimRight(public, "/") + "/telegram/webhook"

		if err := client.SetWebhook(ctx, webhookURL, cfg.TelegramWebhookSecret); err != nil {
			return err
		}
		fmt.Printf("webhook terdaftar: %s\n", webhookURL)

		info, err := client.Info(ctx)
		if err == nil {
			cetakInfo(info)
		}
		return nil

	case "info":
		info, err := client.Info(ctx)
		if err != nil {
			return err
		}
		cetakInfo(info)
		return nil

	default:
		return fmt.Errorf("perintah %q tidak dikenal, pakai \"set\" atau \"info\"", perintah)
	}
}

func cetakInfo(info telegram.WebhookInfo) {
	fmt.Println("status webhook:")
	if info.URL == "" {
		fmt.Println("  url: (belum ada)")
	} else {
		fmt.Printf("  url: %s\n", info.URL)
	}
	fmt.Printf("  update menunggu diproses: %d\n", info.PendingUpdateCount)
	if info.LastErrorMessage != "" {
		fmt.Printf("  error terakhir: %s\n", info.LastErrorMessage)
		fmt.Println("  kalau error menyebut 401/403, cek TELEGRAM_WEBHOOK_SECRET di server dan di sini harus sama.")
	}
}
