// Package telegram membungkus pemanggilan Bot API yang dibutuhkan ANABUL:
// mendaftarkan webhook dan membaca statusnya.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL adalah alamat Bot API Telegram.
const DefaultBaseURL = "https://api.telegram.org"

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

// WebhookInfo adalah isi getWebhookInfo.
type WebhookInfo struct {
	URL                string `json:"url"`
	HasCustomCert      bool   `json:"has_custom_certificate"`
	PendingUpdateCount int    `json:"pending_update_count"`
	LastErrorDate      int64  `json:"last_error_date"`
	LastErrorMessage   string `json:"last_error_message"`
}

// SetWebhook mendaftarkan URL webhook beserta secret token.
func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	if c.token == "" {
		return fmt.Errorf("TELEGRAM_TOKEN belum diisi")
	}
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL webhook Telegram wajib HTTPS, dapat %q", url)
	}
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("TELEGRAM_WEBHOOK_SECRET belum diisi")
	}

	payload := map[string]any{
		"url":                  url,
		"secret_token":         secret,
		"allowed_updates":      []string{"message"},
		"drop_pending_updates": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.call(ctx, "setWebhook", body)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram menolak setWebhook: %s", resp.Description)
	}
	return nil
}

// Info membaca status webhook yang terdaftar.
func (c *Client) Info(ctx context.Context) (WebhookInfo, error) {
	if c.token == "" {
		return WebhookInfo{}, fmt.Errorf("TELEGRAM_TOKEN belum diisi")
	}
	resp, err := c.call(ctx, "getWebhookInfo", []byte("{}"))
	if err != nil {
		return WebhookInfo{}, err
	}
	if !resp.OK {
		return WebhookInfo{}, fmt.Errorf("telegram menolak getWebhookInfo: %s", resp.Description)
	}
	var info WebhookInfo
	if err := json.Unmarshal(resp.Result, &info); err != nil {
		return WebhookInfo{}, err
	}
	return info, nil
}

func (c *Client) call(ctx context.Context, method string, body []byte) (apiResponse, error) {
	endpoint := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return apiResponse{}, fmt.Errorf("panggil telegram: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return apiResponse{}, err
	}

	var parsed apiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return apiResponse{}, fmt.Errorf("respons telegram tidak terbaca (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK && parsed.Description == "" {
		return apiResponse{}, fmt.Errorf("telegram membalas HTTP %d", resp.StatusCode)
	}
	return parsed, nil
}
