package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetWebhook(t *testing.T) {
	var received map[string]any
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "123:ABC")
	if err := client.SetWebhook(context.Background(), "https://anabul.example.com/telegram/webhook", "secret"); err != nil {
		t.Fatalf("SetWebhook error: %v", err)
	}

	if path != "/bot123:ABC/setWebhook" {
		t.Fatalf("path = %q, want /bot123:ABC/setWebhook", path)
	}
	if received["url"] != "https://anabul.example.com/telegram/webhook" {
		t.Fatalf("url sent = %v", received["url"])
	}
	if received["secret_token"] != "secret" {
		t.Fatalf("secret_token sent = %v", received["secret_token"])
	}
	if _, ok := received["allowed_updates"]; !ok {
		t.Fatal("allowed_updates must be sent too")
	}
}

func TestSetWebhookRejectsNonHTTPSURL(t *testing.T) {
	client := NewClient("https://api.telegram.org", "123:ABC")
	err := client.SetWebhook(context.Background(), "http://localhost:8080/telegram/webhook", "secret")
	if err == nil {
		t.Fatal("a non-HTTPS URL must be rejected before calling Telegram")
	}
}

func TestSetWebhookRequiresSecret(t *testing.T) {
	client := NewClient("https://api.telegram.org", "123:ABC")
	if err := client.SetWebhook(context.Background(), "https://example.com/telegram/webhook", "  "); err == nil {
		t.Fatal("an empty secret must be rejected")
	}
}

func TestInfoReadsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"url":"https://anabul.example.com/telegram/webhook","pending_update_count":2,"last_error_message":"Wrong response from the webhook: 401 Unauthorized"}}`))
	}))
	defer server.Close()

	info, err := NewClient(server.URL, "123:ABC").Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.PendingUpdateCount != 2 || info.URL == "" {
		t.Fatalf("info not parsed: %+v", info)
	}
}

func TestEmptyTokenRejected(t *testing.T) {
	client := NewClient("https://api.telegram.org", "")
	if err := client.SetWebhook(context.Background(), "https://example.com/telegram/webhook", "x"); err == nil {
		t.Fatal("an empty token must be rejected")
	}
}
