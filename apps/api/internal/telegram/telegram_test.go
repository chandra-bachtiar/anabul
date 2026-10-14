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
	var terima map[string]any
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &terima)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "123:ABC")
	if err := client.SetWebhook(context.Background(), "https://anabul.contoh.com/telegram/webhook", "rahasia"); err != nil {
		t.Fatalf("SetWebhook error: %v", err)
	}

	if path != "/bot123:ABC/setWebhook" {
		t.Fatalf("path = %q, ingin /bot123:ABC/setWebhook", path)
	}
	if terima["url"] != "https://anabul.contoh.com/telegram/webhook" {
		t.Fatalf("url terkirim = %v", terima["url"])
	}
	if terima["secret_token"] != "rahasia" {
		t.Fatalf("secret_token terkirim = %v", terima["secret_token"])
	}
	if _, ok := terima["allowed_updates"]; !ok {
		t.Fatal("allowed_updates harus ikut dikirim")
	}
}

func TestSetWebhookMenolakURLNonHTTPS(t *testing.T) {
	client := NewClient("https://api.telegram.org", "123:ABC")
	err := client.SetWebhook(context.Background(), "http://localhost:8080/telegram/webhook", "rahasia")
	if err == nil {
		t.Fatal("URL non-HTTPS harus ditolak sebelum memanggil Telegram")
	}
}

func TestSetWebhookButuhSecret(t *testing.T) {
	client := NewClient("https://api.telegram.org", "123:ABC")
	if err := client.SetWebhook(context.Background(), "https://contoh.com/telegram/webhook", "  "); err == nil {
		t.Fatal("secret kosong harus ditolak")
	}
}

func TestInfoMembacaStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"url":"https://anabul.contoh.com/telegram/webhook","pending_update_count":2,"last_error_message":"Wrong response from the webhook: 401 Unauthorized"}}`))
	}))
	defer server.Close()

	info, err := NewClient(server.URL, "123:ABC").Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.PendingUpdateCount != 2 || info.URL == "" {
		t.Fatalf("info tidak terbaca: %+v", info)
	}
}

func TestTokenKosongDitolak(t *testing.T) {
	client := NewClient("https://api.telegram.org", "")
	if err := client.SetWebhook(context.Background(), "https://contoh.com/telegram/webhook", "x"); err == nil {
		t.Fatal("token kosong harus ditolak")
	}
}
