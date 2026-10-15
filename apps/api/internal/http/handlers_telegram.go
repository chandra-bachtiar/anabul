package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type telegramUpdate struct {
	Message *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From *struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"message"`
}

// handleTelegramWebhook takes bot messages. It carries no session cookie, so it
// is verified with the secret token header and the owner chat id.
func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.TelegramToken == "" || s.cfg.TelegramWebhookSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bot Telegram belum dikonfigurasi"})
		return
	}
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != s.cfg.TelegramWebhookSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "secret token tidak cocok"})
		return
	}

	var update telegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update tidak valid"})
		return
	}
	if update.Message == nil || strings.TrimSpace(update.Message.Text) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "skipped": "tanpa teks"})
		return
	}

	chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
	if s.cfg.TelegramOwnerChatID != "" && chatID != strings.TrimSpace(s.cfg.TelegramOwnerChatID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "chat ini tidak diizinkan"})
		return
	}
	if s.chat == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent belum aktif"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	reply, err := s.chat.Chat(ctx, "telegram", update.Message.Text)
	if err != nil {
		slog.Error("failed to process the telegram message", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "gagal memproses pesan"})
		return
	}
	if err := s.sendTelegram(ctx, chatID, reply.Reply); err != nil {
		slog.Error("failed to send the telegram reply", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "gagal mengirim balasan"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) sendTelegram(ctx context.Context, chatID, text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	url := "https://api.telegram.org/bot" + s.cfg.TelegramToken + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram answered %d", resp.StatusCode)
	}
	return nil
}
