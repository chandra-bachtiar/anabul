package http

import (
	"net/http"
	"strconv"
	"time"

	"anabul/internal/tools"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := s.tools.Dashboard(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.tools.MonthSummary(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := tools.TransactionFilter{
		Month: query.Get("month"),
		Query: query.Get("q"),
		Limit: 50,
	}
	if raw := query.Get("category_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, tools.NewInvalidError("category_id harus angka"))
			return
		}
		filter.CategoryID = &id
	}
	if raw := query.Get("cursor"); raw != "" {
		cursor, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, tools.NewInvalidError("cursor harus angka"))
			return
		}
		filter.Cursor = &cursor
	}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || limit <= 0 {
			writeError(w, tools.NewInvalidError("limit harus angka positif"))
			return
		}
		filter.Limit = int32(limit)
	}

	rows, err := s.tools.ListTransactions(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	var nextCursor *int64
	if int32(len(rows)) == filter.Limit && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		nextCursor = &last
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transactions": rows,
		"next_cursor":  nextCursor,
	})
}

type transactionRequest struct {
	Type       string  `json:"type"`
	Amount     int64   `json:"amount"`
	Note       string  `json:"note"`
	CategoryID *int64  `json:"category_id"`
	AccountID  *int64  `json:"account_id"`
	OccurredAt *string `json:"occurred_at"`
}

func (s *Server) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var payload transactionRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	occurredAt, ok := parseOccurredAt(w, payload.OccurredAt)
	if !ok {
		return
	}
	tx, err := s.tools.AddTransaction(r.Context(), tools.AddTransactionInput{
		Type:       payload.Type,
		Amount:     payload.Amount,
		Note:       payload.Note,
		CategoryID: payload.CategoryID,
		AccountID:  payload.AccountID,
		OccurredAt: occurredAt,
		Source:     "web",
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

type transactionPatch struct {
	Type          *string `json:"type"`
	Amount        *int64  `json:"amount"`
	Note          *string `json:"note"`
	CategoryID    *int64  `json:"category_id"`
	ClearCategory bool    `json:"clear_category"`
	AccountID     *int64  `json:"account_id"`
	ClearAccount  bool    `json:"clear_account"`
	OccurredAt    *string `json:"occurred_at"`
}

func (s *Server) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id transaksi tidak valid"))
		return
	}
	var payload transactionPatch
	if !decodeJSON(w, r, &payload) {
		return
	}
	occurredAt, ok := parseOccurredAt(w, payload.OccurredAt)
	if !ok {
		return
	}
	tx, err := s.tools.UpdateTransaction(r.Context(), tools.UpdateTransactionInput{
		ID:            id,
		Type:          payload.Type,
		Amount:        payload.Amount,
		Note:          payload.Note,
		CategoryID:    payload.CategoryID,
		ClearCategory: payload.ClearCategory,
		AccountID:     payload.AccountID,
		ClearAccount:  payload.ClearAccount,
		OccurredAt:    occurredAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (s *Server) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id transaksi tidak valid"))
		return
	}
	if err := s.tools.DeleteTransaction(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.tools.ListCategories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": rows})
}

type categoryRequest struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var payload categoryRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	category, err := s.tools.CreateCategory(r.Context(), payload.Name, payload.Keywords)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kategori tidak valid"))
		return
	}
	var payload struct {
		Name     *string   `json:"name"`
		Keywords *[]string `json:"keywords"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	category, err := s.tools.UpdateCategory(r.Context(), id, payload.Name, payload.Keywords)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kategori tidak valid"))
		return
	}
	if err := s.tools.DeleteCategory(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.tools.ListAccounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": rows})
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	account, err := s.tools.CreateAccount(r.Context(), payload.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func parseOccurredAt(w http.ResponseWriter, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		writeError(w, tools.NewInvalidError("occurred_at harus format RFC3339, mis. 2026-10-13T20:15:00+07:00"))
		return nil, false
	}
	return &parsed, true
}
