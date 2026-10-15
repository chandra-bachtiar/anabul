// Package alert sends error notifications to a webhook (e.g. a Telegram bot or
// Discord) so failures are visible without opening the server logs.
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

// minInterval throttles repeated sends of the same message.
const minInterval = 5 * time.Minute

type Notifier struct {
	url    string
	client *http.Client

	mu       sync.Mutex
	lastSent map[string]time.Time
}

// New returns a notifier; an empty url disables every send and makes Enabled()
// report false.
func New(url string) *Notifier {
	return &Notifier{
		url:      url,
		client:   &http.Client{Timeout: 10 * time.Second},
		lastSent: make(map[string]time.Time),
	}
}

func (n *Notifier) Enabled() bool {
	return n != nil && n.url != ""
}

// Send posts message, skipping repeats within minInterval. It never returns an
// error to the caller: a failed alert must not disturb the app, only be logged.
func (n *Notifier) Send(ctx context.Context, message string) {
	if !n.Enabled() {
		return
	}
	if !n.shouldSend(message) {
		return
	}

	payload, err := json.Marshal(map[string]string{
		"text":       "[ANABUL] " + message,
		"anabul_app": "anabul",
	})
	if err != nil {
		slog.Error("failed to build alert payload", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(payload))
	if err != nil {
		slog.Error("failed to create alert request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Error("failed to send alert", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Error("alert webhook rejected the message", "status", resp.StatusCode)
	}
}

// shouldSend records the last send time per message and drops old entries so
// the map cannot grow forever.
func (n *Notifier) shouldSend(message string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	if last, ok := n.lastSent[message]; ok && now.Sub(last) < minInterval {
		return false
	}
	n.lastSent[message] = now

	for key, sentAt := range n.lastSent {
		if now.Sub(sentAt) > minInterval {
			delete(n.lastSent, key)
		}
	}
	return true
}

// ServerError formats the standard alert text for a 5xx response.
func ServerError(method, path string, status int) string {
	return fmt.Sprintf("error %d on %s %s", status, method, path)
}
