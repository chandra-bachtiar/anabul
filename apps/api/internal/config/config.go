package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config adalah seluruh konfigurasi aplikasi yang dibaca dari environment.
type Config struct {
	Port        int
	Timezone    string
	DatabaseURL string

	AppPasswordHash string
	SessionSecret   string
	// SessionSecretPrevious menerima cookie yang ditandatangani dengan secret
	// lama, dipakai saat rotasi SESSION_SECRET tanpa memaksa login ulang.
	SessionSecretPrevious string

	// TrustProxy menandai server berada di belakang reverse proxy, sehingga
	// header X-Forwarded-For dan X-Forwarded-Proto boleh dipercaya.
	TrustProxy bool

	// AlertWebhookURL menerima notifikasi singkat saat terjadi error 5xx
	// atau kegagalan start. Kosong = notifikasi dimatikan.
	AlertWebhookURL string

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

	trustProxy, err := envBool("TRUST_PROXY", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:        port,
		Timezone:    tz,
		DatabaseURL: env("DATABASE_URL", "postgres://app:app@localhost:5432/finance?sslmode=disable"),

		AppPasswordHash:       os.Getenv("APP_PASSWORD_HASH"),
		SessionSecret:         os.Getenv("SESSION_SECRET"),
		SessionSecretPrevious: os.Getenv("SESSION_SECRET_PREVIOUS"),
		TrustProxy:            trustProxy,
		AlertWebhookURL:       os.Getenv("ALERT_WEBHOOK_URL"),

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

func envBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s %q bukan true/false", key, raw)
	}
	return value, nil
}
