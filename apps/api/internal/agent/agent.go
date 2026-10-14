// Package agent berisi loop LLM tool calling, registry tool, dan penyusunan
// konteks untuk ANABUL.
package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"anabul/internal/store"
	"anabul/internal/tools"
)

const (
	maxTurns          = 5
	historyLimit      = 8
	pendingTTL        = 15 * time.Minute
	fallbackModelName = "parser-sederhana"
)

// Reply adalah hasil percakapan yang dikirim balik ke klien.
type Reply struct {
	Reply         string   `json:"reply"`
	PendingAction *Pending `json:"pending_action,omitempty"`
	Undo          *Undo    `json:"undo,omitempty"`
	Source        string   `json:"source,omitempty"`
}

// Undo memberi UI kesempatan membatalkan transaksi yang baru dicatat.
type Undo struct {
	TransactionID int64  `json:"transaction_id"`
	Label         string `json:"label,omitempty"`
}

// Pending adalah aksi berisiko yang menunggu konfirmasi pengguna.
type Pending struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// Config berisi pengaturan klien LLM.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Agent menjalankan percakapan: membangun konteks, memanggil LLM, menjalankan
// tool, dan mencatat semuanya ke agent_logs.
type Agent struct {
	cfg   Config
	tools *tools.Service
	q     *store.Queries
	llm   *llmClient
}

// New membuat agent. Klien LLM hanya aktif kalau API key dan base URL terisi;
// tanpa itu agent memakai parser sederhana.
func New(cfg Config, svc *tools.Service, q *store.Queries) *Agent {
	agent := &Agent{cfg: cfg, tools: svc, q: q}
	if strings.TrimSpace(cfg.APIKey) != "" && strings.TrimSpace(cfg.BaseURL) != "" {
		agent.llm = newLLMClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	}
	return agent
}

// LLMEnabled menandai apakah agent memakai model atau parser sederhana.
func (a *Agent) LLMEnabled() bool { return a.llm != nil }

// Chat memproses satu pesan pengguna dari kanal tertentu (web atau telegram).
func (a *Agent) Chat(ctx context.Context, channel, message string) (Reply, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return Reply{}, tools.NewInvalidError("pesannya masih kosong")
	}

	channel = normalizeChannel(channel)
	ctx = withChannel(ctx, channel)
	if err := a.q.InsertChatMessage(ctx, store.InsertChatMessageParams{
		Channel: channel,
		Role:    "user",
		Content: message,
	}); err != nil {
		return Reply{}, err
	}

	var (
		reply   Reply
		err     error
		usedLLM = a.llm != nil
	)
	if usedLLM {
		reply, err = a.chatWithLLM(ctx, channel, message)
	} else {
		reply, err = a.chatWithParser(ctx, channel, message)
	}
	if err != nil {
		reply = Reply{Reply: friendlyError(err), Source: sourceOf(usedLLM)}
	}

	if err := a.q.InsertChatMessage(ctx, store.InsertChatMessageParams{
		Channel: channel,
		Role:    "assistant",
		Content: reply.Reply,
	}); err != nil {
		slog.Error("gagal menyimpan balasan chat", "error", err)
	}
	return reply, nil
}

func (a *Agent) chatWithLLM(ctx context.Context, channel, message string) (Reply, error) {
	system := a.systemPrompt(ctx, channel)
	messages := []chatMessage{{Role: "system", Content: system}}
	messages = append(messages, trimHistory(a.historyMessages(ctx, channel, historyLimit+1), historyLimit)...)
	messages = append(messages, chatMessage{Role: "user", Content: message})

	schemas := a.toolSchemas()
	toolLog := make([]map[string]any, 0, 4)
	var pending *Pending

	for turn := 1; turn <= maxTurns; turn++ {
		response, err := a.llm.complete(ctx, messages, schemas)
		if err != nil {
			return Reply{}, err
		}

		if len(response.ToolCalls) == 0 {
			reply := strings.TrimSpace(response.Content)
			if reply == "" {
				reply = "Meong, aku belum bisa menjawab itu. Coba tulis lagi ya."
			}
			a.log(ctx, channel, message, reply, toolLog)
			return Reply{Reply: reply, PendingAction: pending, Undo: undoFromLog(toolLog), Source: sourceOf(true)}, nil
		}

		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		for _, call := range response.ToolCalls {
			args := json.RawMessage(call.Function.Arguments)
			if len(args) == 0 {
				args = json.RawMessage("{}")
			}
			result, runErr := a.runTool(ctx, call.Function.Name, args, false)
			if runErr != nil {
				return Reply{}, runErr
			}
			toolLog = append(toolLog, map[string]any{
				"name":   call.Function.Name,
				"args":   args,
				"result": result,
			})
			if id, summary, ok := confirmationOf(result); ok {
				pending = &Pending{ID: id, Summary: summary}
			}
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    result,
			})
		}
	}

	reply := "Maaf, aku belum selesai memprosesnya. Coba tulis lebih sederhana ya, misalnya \"kopi 25rb\"."
	a.log(ctx, channel, message, reply, toolLog)
	return Reply{Reply: reply, PendingAction: pending, Undo: undoFromLog(toolLog), Source: sourceOf(true)}, nil
}

func (a *Agent) chatWithParser(ctx context.Context, channel, message string) (Reply, error) {
	reply, toolLog, err := a.runParser(ctx, channel, message)
	if err != nil {
		return Reply{}, err
	}
	a.log(ctx, channel, message, reply, toolLog)
	return Reply{Reply: reply, Undo: undoFromLog(toolLog), Source: sourceOf(false)}, nil
}

// undoFromLog mencari hasil add_transaction supaya UI bisa menawarkan Undo.
func undoFromLog(toolLog []map[string]any) *Undo {
	for i := len(toolLog) - 1; i >= 0; i-- {
		if name, _ := toolLog[i]["name"].(string); name != "add_transaction" {
			continue
		}
		switch result := toolLog[i]["result"].(type) {
		case string:
			var parsed struct {
				ID   int64  `json:"id"`
				Note string `json:"note"`
			}
			if err := json.Unmarshal([]byte(result), &parsed); err == nil && parsed.ID > 0 {
				return &Undo{TransactionID: parsed.ID, Label: parsed.Note}
			}
		case tools.Transaction:
			if result.ID > 0 {
				return &Undo{TransactionID: result.ID, Label: result.Note}
			}
		}
	}
	return nil
}

// Confirm menjalankan aksi berisiko yang sebelumnya ditahan.
func (a *Agent) Confirm(ctx context.Context, channel, actionID string) (Reply, error) {
	action, err := a.loadPending(ctx, actionID)
	if err != nil {
		return Reply{}, err
	}
	if err := a.q.DeletePendingAction(ctx, action.ID); err != nil {
		return Reply{}, err
	}

	def, found := a.toolByName(action.Tool)
	if !found {
		return Reply{}, tools.NewNotFoundError("aksi %s tidak dikenal lagi", action.Tool)
	}
	raw, err := def.Run(ctx, json.RawMessage(action.Args))
	if err != nil {
		return Reply{Reply: friendlyError(err), Source: sourceOf(a.llm != nil)}, nil
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return Reply{}, err
	}
	reply := describeResult(action.Tool, encoded)
	a.log(ctx, channel, "(konfirmasi "+action.Tool+")", reply, []map[string]any{{
		"name":   action.Tool,
		"args":   json.RawMessage(action.Args),
		"result": string(encoded),
	}})
	if err := a.q.InsertChatMessage(ctx, store.InsertChatMessageParams{
		Channel: normalizeChannel(channel),
		Role:    "assistant",
		Content: reply,
	}); err != nil {
		slog.Error("gagal menyimpan balasan konfirmasi", "error", err)
	}
	return Reply{Reply: reply, Source: sourceOf(a.llm != nil)}, nil
}

// Cancel membatalkan aksi berisiko yang menunggu konfirmasi.
func (a *Agent) Cancel(ctx context.Context, actionID string) (Reply, error) {
	action, err := a.loadPending(ctx, actionID)
	if err != nil {
		return Reply{}, err
	}
	if err := a.q.DeletePendingAction(ctx, action.ID); err != nil {
		return Reply{}, err
	}
	return Reply{Reply: "Oke, dibatalkan. Tidak ada data yang berubah."}, nil
}

func (a *Agent) createPending(ctx context.Context, def toolDef, args json.RawMessage) (map[string]any, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	if err := a.q.CreatePendingAction(ctx, store.CreatePendingActionParams{
		ID:        id,
		Tool:      def.Name,
		Args:      args,
		ExpiresAt: pgtype.Timestamptz{Time: timeNow(a.tools.Location()).Add(pendingTTL), Valid: true},
	}); err != nil {
		return nil, err
	}
	return map[string]any{
		"needs_confirmation": true,
		"action_id":          id,
		"summary":            a.riskSummary(def.Name, args),
	}, nil
}

func (a *Agent) loadPending(ctx context.Context, actionID string) (store.PendingAction, error) {
	action, err := a.q.GetPendingAction(ctx, strings.TrimSpace(actionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.PendingAction{}, tools.NewNotFoundError("aksi itu tidak ditemukan atau sudah dipakai")
		}
		return store.PendingAction{}, err
	}
	if action.ExpiresAt.Valid && timeNow(a.tools.Location()).After(action.ExpiresAt.Time) {
		_ = a.q.DeletePendingAction(ctx, action.ID)
		return store.PendingAction{}, tools.NewNotFoundError("aksinya sudah kedaluwarsa, ulangi perintahnya ya")
	}
	return action, nil
}

func (a *Agent) riskSummary(name string, args json.RawMessage) string {
	var parsed struct {
		Month  string `json:"month"`
		ID     *int64 `json:"id"`
		Last   bool   `json:"last"`
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}
	_ = json.Unmarshal(args, &parsed)

	switch name {
	case "close_month":
		month := parsed.Month
		if month == "" {
			month = tools.FormatMonth(a.tools.CurrentMonth())
		}
		return "Tutup bulan " + month + "? Setelah ditutup, transaksi bulan itu tidak bisa diubah."
	case "reopen_month":
		return "Buka kembali bulan " + parsed.Month + " supaya bisa diedit?"
	case "delete_transaction":
		if parsed.ID != nil {
			return fmt.Sprintf("Hapus transaksi #%d?", *parsed.ID)
		}
		if parsed.Last {
			return "Hapus transaksi terakhir?"
		}
		return "Hapus transaksi itu?"
	default:
		return "Jalankan aksi " + name + "?"
	}
}

func (a *Agent) log(ctx context.Context, channel, userMessage, reply string, toolCalls []map[string]any) {
	payload, err := json.Marshal(toolCalls)
	if err != nil {
		payload = []byte("[]")
	}
	if err := a.q.InsertAgentLog(ctx, store.InsertAgentLogParams{
		Channel:     normalizeChannel(channel),
		UserMessage: userMessage,
		ToolCalls:   payload,
		Reply:       pgtype.Text{String: reply, Valid: reply != ""},
	}); err != nil {
		slog.Error("gagal menulis agent_logs", "error", err)
	}
}

func confirmationOf(result string) (string, string, bool) {
	var parsed struct {
		NeedsConfirmation bool   `json:"needs_confirmation"`
		ActionID          string `json:"action_id"`
		Summary           string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return "", "", false
	}
	if !parsed.NeedsConfirmation || parsed.ActionID == "" {
		return "", "", false
	}
	return parsed.ActionID, parsed.Summary, true
}

func normalizeChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "telegram":
		return "telegram"
	default:
		return "web"
	}
}

type channelKey struct{}

// withChannel menempelkan kanal percakapan ke context supaya tool mencatat
// sumber transaksi dengan benar (web atau telegram).
func withChannel(ctx context.Context, channel string) context.Context {
	return context.WithValue(ctx, channelKey{}, normalizeChannel(channel))
}

func channelFrom(ctx context.Context) string {
	if value, ok := ctx.Value(channelKey{}).(string); ok && value != "" {
		return value
	}
	return "web"
}

func sourceOf(usedLLM bool) string {
	if usedLLM {
		return "llm"
	}
	return fallbackModelName
}

func randomID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// describeResult menyusun balasan manusiawi setelah aksi berisiko dijalankan.
func describeResult(toolName string, result json.RawMessage) string {
	switch toolName {
	case "close_month":
		var parsed struct {
			Month   string `json:"month"`
			Income  int64  `json:"income"`
			Expense int64  `json:"expense"`
			Net     int64  `json:"net"`
			Budgets int    `json:"carried_budgets"`
			Needs   int    `json:"carried_needs"`
		}
		if err := json.Unmarshal(result, &parsed); err == nil {
			return fmt.Sprintf("Bulan %s sudah ditutup meong.\nPemasukan %s, pengeluaran %s, selisih %s.\nBulan berikutnya sudah disiapkan: %d budget dan %d kebutuhan.",
				parsed.Month, tools.FormatAmount(parsed.Income), tools.FormatAmount(parsed.Expense),
				tools.FormatAmount(parsed.Net), parsed.Budgets, parsed.Needs)
		}
	case "reopen_month":
		var parsed struct {
			Month string `json:"month"`
		}
		if err := json.Unmarshal(result, &parsed); err == nil {
			return fmt.Sprintf("Bulan %s sudah dibuka kembali. Transaksi bulan itu bisa diubah lagi.", parsed.Month)
		}
	case "delete_transaction":
		var parsed struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(result, &parsed); err == nil {
			return fmt.Sprintf("Transaksi #%d sudah dihapus.", parsed.ID)
		}
	}
	return "Selesai, sudah aku jalankan."
}

func friendlyError(err error) string {
	if domainErr := tools.AsError(err); domainErr != nil {
		if len(domainErr.Available) > 0 {
			return domainErr.Message + "\nPilihan yang ada: " + strings.Join(domainErr.Available, ", ")
		}
		return domainErr.Message
	}
	slog.Error("agent gagal memproses pesan", "error", err)
	return "Maaf, ada gangguan teknis saat memproses pesanmu. Coba lagi sebentar lagi ya."
}

func storeListRecentParams(channel string, limit int32) store.ListRecentChatMessagesParams {
	return store.ListRecentChatMessagesParams{Channel: normalizeChannel(channel), Limit: limit}
}
