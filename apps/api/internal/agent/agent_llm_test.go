package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/store"
	"anabul/internal/tools"
	"anabul/migrations"
)

// TestChatWithMockLLM covers the tool-calling loop end to end: a fake model
// answers with add_transaction, the agent runs it through the tools layer, then
// the fake model writes the final reply. Needs a test database; skipped without
// ANABUL_TEST_DATABASE_URL so `make test` still works offline.
func TestChatWithMockLLM(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ANABUL_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ANABUL_TEST_DATABASE_URL is empty, skipping the LLM loop test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to the test database: %v", err)
	}
	// t.Cleanup runs LIFO: registering the pool close first makes the data
	// cleanup run while the connection is still open.
	t.Cleanup(pool.Close)

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrate the test database: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM transactions WHERE note = $1", "kopi uji"); err != nil {
		t.Fatalf("clear test rows: %v", err)
	}

	var logStart, chatStart int64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM agent_logs").Scan(&logStart)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM chat_messages").Scan(&chatStart)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE note = $1", "kopi uji")
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM agent_logs WHERE id > $1", logStart)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM chat_messages WHERE id > $1", chatStart)
	})

	queries := store.New(pool)
	service := tools.New(queries, pool, time.FixedZone("WIB", 7*60*60))

	// Fake model: the first round asks for add_transaction, the second replies.
	var calls int
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls++

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"role":"tool"`) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Pengeluaran Rp 25.000 buat kopi uji tercatat."}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"add_transaction","arguments":"{\"amount\":25000,\"note\":\"kopi uji\",\"type\":\"expense\"}"}}]}}]}`))
	}))
	defer llm.Close()

	chatbot := New(Config{BaseURL: llm.URL, APIKey: "test-key", Model: "fake-model"}, service, queries)
	if !chatbot.LLMEnabled() {
		t.Fatal("the agent must use the LLM client when a key is set")
	}

	reply, err := chatbot.Chat(ctx, "web", "kopi uji 25rb")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if !strings.Contains(reply.Reply, "tercatat") {
		t.Fatalf("unexpected reply: %q", reply.Reply)
	}
	if calls < 2 {
		t.Fatalf("the loop must call the model twice, got %d", calls)
	}
	if reply.Undo == nil || reply.Undo.TransactionID == 0 {
		t.Fatalf("the agent must offer undo for the new transaction, got %+v", reply.Undo)
	}

	last, err := service.LastTransaction(ctx)
	if err != nil {
		t.Fatalf("read the last transaction: %v", err)
	}
	if last.Amount != 25000 || last.Note != "kopi uji" {
		t.Fatalf("the transaction was not stored correctly: %+v", last)
	}

	// A risky action must not run inline: it becomes a pending action.
	riskyLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_2","type":"function","function":{"name":"close_month","arguments":"{}"}}]}}]}`))
	}))
	defer riskyLLM.Close()

	riskyChatbot := New(Config{BaseURL: riskyLLM.URL, APIKey: "test-key", Model: "fake-model"}, service, queries)
	riskyReply, err := riskyChatbot.Chat(ctx, "web", "tutup bulan sekarang")
	if err != nil {
		t.Fatalf("chat with a risky action: %v", err)
	}
	if riskyReply.PendingAction == nil || riskyReply.PendingAction.ID == "" {
		t.Fatalf("a risky action must wait for confirmation, got %+v", riskyReply)
	}

	// Confirming runs the action and clears the pending row.
	confirmed, err := riskyChatbot.Confirm(ctx, "web", riskyReply.PendingAction.ID)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if !strings.Contains(confirmed.Reply, "ditutup") {
		t.Fatalf("unexpected confirmation reply: %q", confirmed.Reply)
	}

	// Restore the state so the test can run again.
	month := tools.FormatMonth(service.CurrentMonth())
	if err := service.ReopenMonth(ctx, month); err != nil {
		t.Fatalf("reopen the test month: %v", err)
	}

	payload, _ := json.Marshal(riskyReply.PendingAction)
	t.Logf("pending action: %s", payload)
}
