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

// TestChatWithMockLLM adalah test regresi loop tool calling: model tiruan
// mengembalikan tool_call add_transaction, agent menjalankannya lewat layer
// tools, lalu membalas. Butuh database uji; tanpa env ANABUL_TEST_DATABASE_URL
// test ini dilewati supaya `make test` tetap jalan tanpa DB.
func TestChatWithMockLLM(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ANABUL_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ANABUL_TEST_DATABASE_URL belum diisi, test loop LLM dilewati")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("koneksi database uji: %v", err)
	}
	// t.Cleanup berjalan terbalik (LIFO): tutup pool didaftarkan lebih dulu
	// supaya pembersihan data berjalan sebelum koneksi ditutup.
	t.Cleanup(pool.Close)

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrasi database uji: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM transactions WHERE note = $1", "kopi uji"); err != nil {
		t.Fatalf("bersihkan data uji: %v", err)
	}

	var logAwal, chatAwal int64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM agent_logs").Scan(&logAwal)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM chat_messages").Scan(&chatAwal)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE note = $1", "kopi uji")
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM agent_logs WHERE id > $1", logAwal)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM chat_messages WHERE id > $1", chatAwal)
	})

	queries := store.New(pool)
	service := tools.New(queries, pool, time.FixedZone("WIB", 7*60*60))

	// Model tiruan: putaran pertama minta add_transaction, putaran kedua membalas.
	var panggilan int
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		panggilan++

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"role":"tool"`) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Pengeluaran Rp 25.000 buat kopi uji tercatat."}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"add_transaction","arguments":"{\"amount\":25000,\"note\":\"kopi uji\",\"type\":\"expense\"}"}}]}}]}`))
	}))
	defer llm.Close()

	chatbot := New(Config{BaseURL: llm.URL, APIKey: "kunci-uji", Model: "model-tiruan"}, service, queries)
	if !chatbot.LLMEnabled() {
		t.Fatal("agent harus memakai klien LLM saat key diisi")
	}

	balasan, err := chatbot.Chat(ctx, "web", "kopi uji 25rb")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if !strings.Contains(balasan.Reply, "tercatat") {
		t.Fatalf("balasan tidak sesuai: %q", balasan.Reply)
	}
	if panggilan < 2 {
		t.Fatalf("loop harus memanggil model dua kali, dapat %d", panggilan)
	}
	if balasan.Undo == nil || balasan.Undo.TransactionID == 0 {
		t.Fatalf("agent harus menawarkan undo untuk transaksi baru, dapat %+v", balasan.Undo)
	}

	terakhir, err := service.LastTransaction(ctx)
	if err != nil {
		t.Fatalf("ambil transaksi terakhir: %v", err)
	}
	if terakhir.Amount != 25000 || terakhir.Note != "kopi uji" {
		t.Fatalf("transaksi tidak tercatat benar: %+v", terakhir)
	}

	// Aksi berisiko tidak boleh langsung jalan: harus jadi pending action.
	llmRisiko := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_2","type":"function","function":{"name":"close_month","arguments":"{}"}}]}}]}`))
	}))
	defer llmRisiko.Close()

	chatbotRisiko := New(Config{BaseURL: llmRisiko.URL, APIKey: "kunci-uji", Model: "model-tiruan"}, service, queries)
	balasanRisiko, err := chatbotRisiko.Chat(ctx, "web", "tutup bulan sekarang")
	if err != nil {
		t.Fatalf("chat aksi berisiko: %v", err)
	}
	if balasanRisiko.PendingAction == nil || balasanRisiko.PendingAction.ID == "" {
		t.Fatalf("aksi berisiko harus menunggu konfirmasi, dapat %+v", balasanRisiko)
	}

	// Konfirmasi menjalankan aksinya dan membersihkan pending action.
	hasil, err := chatbotRisiko.Confirm(ctx, "web", balasanRisiko.PendingAction.ID)
	if err != nil {
		t.Fatalf("konfirmasi: %v", err)
	}
	if !strings.Contains(hasil.Reply, "ditutup") {
		t.Fatalf("balasan konfirmasi tidak sesuai: %q", hasil.Reply)
	}

	// Kembalikan keadaan supaya test bisa dijalankan berulang.
	bulan := tools.FormatMonth(service.CurrentMonth())
	if err := service.ReopenMonth(ctx, bulan); err != nil {
		t.Fatalf("buka kembali bulan uji: %v", err)
	}

	payload, _ := json.Marshal(balasanRisiko.PendingAction)
	t.Logf("pending action: %s", payload)
}
