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

func TestSendDeliversMessages(t *testing.T) {
	var mu sync.Mutex
	var messages []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		messages = append(messages, payload["text"])
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := New(server.URL)
	if !notifier.Enabled() {
		t.Fatal("notifier must be enabled when a url is set")
	}

	ctx := context.Background()
	notifier.Send(ctx, "error 500 on GET /api/x")
	notifier.Send(ctx, "error 500 on GET /api/x") // duplicate, must be throttled
	notifier.Send(ctx, "error 500 on GET /api/y")

	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 2 {
		t.Fatalf("messages sent = %d, want 2 (the duplicate is throttled): %v", len(messages), messages)
	}
	if messages[0] != "[ANABUL] error 500 on GET /api/x" {
		t.Fatalf("first message = %q", messages[0])
	}
}

func TestNotifierDisabledWithoutURL(t *testing.T) {
	notifier := New("")
	if notifier.Enabled() {
		t.Fatal("notifier without a url must be disabled")
	}
	// Must not panic and must not send anything.
	notifier.Send(context.Background(), "anything")
}

func TestMinIntervalApplies(t *testing.T) {
	notifier := New("https://example.invalid/webhook")
	if !notifier.shouldSend("message") {
		t.Fatal("first send must be allowed")
	}
	if notifier.shouldSend("message") {
		t.Fatal("second send within the interval must be throttled")
	}
	// Once the interval has passed, the same message may be sent again.
	notifier.mu.Lock()
	notifier.lastSent["message"] = time.Now().Add(-minInterval - time.Second)
	notifier.mu.Unlock()
	if !notifier.shouldSend("message") {
		t.Fatal("after the interval has passed, the send must be allowed")
	}
}
