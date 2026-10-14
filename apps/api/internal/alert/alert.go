// Package alert mengirim notifikasi error ke webhook (mis. bot Telegram atau
// Discord) supaya gangguan terlihat tanpa harus membuka log server.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// jedaMinimal membatasi pengiriman berulang untuk pesan yang sama.
const jedaMinimal = 5 * time.Minute

type Notifier struct {
	url    string
	client *http.Client

	mu       sync.Mutex
	terakhir map[string]time.Time
}

// New mengembalikan notifier; kalau url kosong, semua kiriman diabaikan dan
// Enabled() bernilai false.
func New(url string) *Notifier {
	return &Notifier{
		url:      url,
		client:   &http.Client{Timeout: 10 * time.Second},
		terakhir: make(map[string]time.Time),
	}
}

func (n *Notifier) Enabled() bool {
	return n != nil && n.url != ""
}

// Kirim mengirim pesan. Pesan yang sama dalam jedaMinimal tidak dikirim ulang.
// Tidak pernah mengembalikan error ke pemanggil: kegagalan alert tidak boleh
// mengganggu aplikasi, cukup dicatat di log.
func (n *Notifier) Kirim(ctx context.Context, pesan string) {
	if !n.Enabled() {
		return
	}
	if !n.bolehKirim(pesan) {
		return
	}

	payload, err := json.Marshal(map[string]string{
		"text":       "[ANABUL] " + pesan,
		"anabul_app": "anabul",
	})
	if err != nil {
		slog.Error("gagal menyiapkan alert", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(payload))
	if err != nil {
		slog.Error("gagal membuat request alert", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Error("gagal mengirim alert", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Error("webhook alert menolak pesan", "status", resp.StatusCode)
	}
}

// bolehKirim mencatat waktu kirim terakhir per pesan.
func (n *Notifier) bolehKirim(pesan string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	if terakhir, ada := n.terakhir[pesan]; ada && now.Sub(terakhir) < jedaMinimal {
		return false
	}
	n.terakhir[pesan] = now

	// Bersihkan catatan lama supaya map tidak tumbuh terus.
	for kunci, waktu := range n.terakhir {
		if now.Sub(waktu) > jedaMinimal {
			delete(n.terakhir, kunci)
		}
	}
	return true
}

// PesanServer menulis pesan alert standar untuk error 5xx.
func PesanServer(method, path string, status int) string {
	return fmt.Sprintf("error %d di %s %s", status, method, path)
}
