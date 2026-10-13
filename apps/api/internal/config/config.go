package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config adalah seluruh konfigurasi aplikasi yang dibaca dari environment.
type Config struct {
	Port        int
	Timezone    string
	DatabaseURL string

	AppPasswordHash string
	SessionSecret   string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string

	TelegramToken         string
	TelegramWebhookSecret string
	TelegramOwnerChatID   string
}

// Load membaca environment dan memvalidasi nilai yang dipakai aplikasi.
func Load() (Config, error) {
	rawPort := env("PORT", "8080")
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return Config{}, fmt.Errorf("PORT %q bukan angka", rawPort)
	}

	tz := env("TZ", "Asia/Jakarta")
	if _, err := time.LoadLocation(tz); err != nil {
		return Config{}, fmt.Errorf("TZ %q tidak dikenal: %w", tz, err)
	}

	return Config{
		Port:        port,
		Timezone:    tz,
		DatabaseURL: env("DATABASE_URL", "postgres://app:app@localhost:5432/finance?sslmode=disable"),

		AppPasswordHash: os.Getenv("APP_PASSWORD_HASH"),
		SessionSecret:   os.Getenv("SESSION_SECRET"),

		LLMBaseURL: env("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMAPIKey:  os.Getenv("LLM_API_KEY"),
		LLMModel:   env("LLM_MODEL", "gpt-4o-mini"),

		TelegramToken:         os.Getenv("TELEGRAM_TOKEN"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		TelegramOwnerChatID:   os.Getenv("TELEGRAM_OWNER_CHAT_ID"),
	}, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
