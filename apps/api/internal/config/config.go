package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the full application configuration read from the environment.
type Config struct {
	Port        int
	Timezone    string
	DatabaseURL string

	AppPasswordHash string
	SessionSecret   string
	// SessionSecretPrevious accepts cookies signed with the old secret, used
	// when rotating SESSION_SECRET without forcing a re-login.
	SessionSecretPrevious string

	// TrustProxy marks the server as running behind a reverse proxy, so the
	// X-Forwarded-For and X-Forwarded-Proto headers may be trusted.
	TrustProxy bool

	// AlertWebhookURL receives short notifications on 5xx errors or a failed
	// start. Empty disables notifications.
	AlertWebhookURL string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string

	TelegramToken         string
	TelegramWebhookSecret string
	TelegramOwnerChatID   string
}

// Load reads the environment and validates the values the app uses.
func Load() (Config, error) {
	rawPort := env("PORT", "8080")
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return Config{}, fmt.Errorf("PORT %q is not a number", rawPort)
	}

	tz := env("TZ", "Asia/Jakarta")
	if _, err := time.LoadLocation(tz); err != nil {
		return Config{}, fmt.Errorf("unknown TZ %q: %w", tz, err)
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
		return false, fmt.Errorf("%s %q is not true/false", key, raw)
	}
	return value, nil
}
