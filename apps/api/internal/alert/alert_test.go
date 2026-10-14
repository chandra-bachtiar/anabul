package alert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestKirimMengirimPesan(t *testing.T) {
	var mu sync.Mutex
	var pesan []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		pesan = append(pesan, payload["text"])
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := New(server.URL)
	if !notifier.Enabled() {
		t.Fatal("notifier harus aktif saat url diisi")
	}

	ctx := context.Background()
	notifier.Kirim(ctx, "error 500 di GET /api/x")
	notifier.Kirim(ctx, "error 500 di GET /api/x") // pesan sama, harus ditahan
	notifier.Kirim(ctx, "error 500 di GET /api/y")

	mu.Lock()
	defer mu.Unlock()
	if len(pesan) != 2 {
		t.Fatalf("pesan terkirim = %d, ingin 2 (yang sama ditahan): %v", len(pesan), pesan)
	}
	if pesan[0] != "[ANABUL] error 500 di GET /api/x" {
		t.Fatalf("pesan pertama = %q", pesan[0])
	}
}

func TestNotifierMatiTanpaURL(t *testing.T) {
	notifier := New("")
	if notifier.Enabled() {
		t.Fatal("notifier tanpa url harus mati")
	}
	// Tidak boleh panik dan tidak mengirim apa pun.
	notifier.Kirim(context.Background(), "apa saja")
}

func TestJedaMinimalBerlaku(t *testing.T) {
	notifier := New("https://contoh.invalid/webhook")
	if !notifier.bolehKirim("pesan") {
		t.Fatal("kiriman pertama harus boleh")
	}
	if notifier.bolehKirim("pesan") {
		t.Fatal("kiriman kedua dalam jeda harus ditahan")
	}
	// Setelah jeda lewat, pesan yang sama boleh dikirim lagi.
	notifier.mu.Lock()
	notifier.terakhir["pesan"] = time.Now().Add(-jedaMinimal - time.Second)
	notifier.mu.Unlock()
	if !notifier.bolehKirim("pesan") {
		t.Fatal("setelah jeda lewat, kiriman harus boleh")
	}
}
