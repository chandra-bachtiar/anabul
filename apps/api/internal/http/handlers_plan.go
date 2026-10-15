package http

import (
	"net/http"
	"time"

	"anabul/internal/tools"
)

func (s *Server) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.tools.ListBudgets(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"budgets": rows})
}

func (s *Server) handleSetBudgets(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Month string `json:"month"`
		Items []struct {
			CategoryID int64 `json:"category_id"`
			Amount     int64 `json:"amount"`
		} `json:"items"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len(payload.Items) == 0 {
		writeError(w, tools.NewInvalidError("items belum diisi"))
		return
	}

	saved := make([]tools.BudgetRow, 0, len(payload.Items))
	for _, item := range payload.Items {
		row, err := s.tools.SetBudget(r.Context(), payload.Month, item.CategoryID, item.Amount)
		if err != nil {
			writeError(w, err)
			return
		}
		saved = append(saved, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"budgets": saved})
}

func (s *Server) handleCopyBudgets(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	count, err := s.tools.CopyBudgets(r.Context(), payload.From, payload.To)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "copied": count})
}

func (s *Server) handleListRecurringNeeds(w http.ResponseWriter, r *http.Request) {
	rows, err := s.tools.ListRecurringNeeds(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurring_needs": rows})
}

func (s *Server) handleCreateRecurringNeed(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name       string `json:"name"`
		Amount     int64  `json:"amount"`
		DueDay     int    `json:"due_day"`
		CategoryID *int64 `json:"category_id"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	row, err := s.tools.CreateRecurringNeed(r.Context(), payload.Name, payload.Amount, payload.DueDay, payload.CategoryID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateRecurringNeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kebutuhan tidak valid"))
		return
	}
	var payload struct {
		Name       *string `json:"name"`
		Amount     *int64  `json:"amount"`
		DueDay     *int    `json:"due_day"`
		CategoryID *int64  `json:"category_id"`
		Active     *bool   `json:"active"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	row, err := s.tools.UpdateRecurringNeed(r.Context(), tools.UpdateRecurringNeedInput{
		ID:         id,
		Name:       payload.Name,
		Amount:     payload.Amount,
		DueDay:     payload.DueDay,
		CategoryID: payload.CategoryID,
		Active:     payload.Active,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteRecurringNeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kebutuhan tidak valid"))
		return
	}
	if err := s.tools.DeleteRecurringNeed(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleListNeeds(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.tools.ListMonthlyNeeds(r.Context(), query.Get("month"), query.Get("status"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"needs": rows})
}

func (s *Server) handleGenerateNeeds(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Month string `json:"month"`
	}
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &payload) {
			return
		}
	}
	count, err := s.tools.GenerateMonthlyNeeds(r.Context(), payload.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "needs": count})
}

func (s *Server) handlePayNeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kebutuhan tidak valid"))
		return
	}
	var payload struct {
		Amount *int64 `json:"amount"`
	}
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &payload) {
			return
		}
	}
	row, err := s.tools.MarkNeedPaid(r.Context(), id, payload.Amount)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleUnpayNeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id kebutuhan tidak valid"))
		return
	}
	row, err := s.tools.UnmarkNeedPaid(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleListDebts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.tools.ListDebts(r.Context(), query.Get("status"), query.Get("person"))
	if err != nil {
		writeError(w, err)
		return
	}
	totals, err := s.tools.DebtTotals(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"debts": rows, "totals": totals})
}

func (s *Server) handleCreateDebt(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Person    string  `json:"person"`
		Direction string  `json:"direction"`
		Amount    int64   `json:"amount"`
		Note      string  `json:"note"`
		DueDate   *string `json:"due_date"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	dueDate, ok := parseDueDate(w, payload.DueDate)
	if !ok {
		return
	}
	debt, err := s.tools.AddDebt(r.Context(), tools.AddDebtInput{
		Person:    payload.Person,
		Direction: payload.Direction,
		Amount:    payload.Amount,
		Note:      payload.Note,
		DueDate:   dueDate,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, debt)
}

func (s *Server) handleGetDebt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id hutang tidak valid"))
		return
	}
	debt, err := s.tools.GetDebt(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, debt)
}

func (s *Server) handleCreateDebtPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, tools.NewInvalidError("id hutang tidak valid"))
		return
	}
	var payload struct {
		Amount            int64   `json:"amount"`
		PaidAt            *string `json:"paid_at"`
		RecordTransaction *bool   `json:"record_transaction"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	paidAt, ok := parseOccurredAt(w, payload.PaidAt)
	if !ok {
		return
	}
	// A payment is booked as a transaction by default so the month's cash flow
	// counts it; record_transaction: false turns that off.
	recordTransaction := true
	if payload.RecordTransaction != nil {
		recordTransaction = *payload.RecordTransaction
	}
	debt, err := s.tools.AddDebtPayment(r.Context(), tools.AddDebtPaymentInput{
		DebtID:            &id,
		Amount:            payload.Amount,
		PaidAt:            paidAt,
		RecordTransaction: recordTransaction,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, debt)
}

func parseDueDate(w http.ResponseWriter, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		writeError(w, tools.NewInvalidError("due_date harus format YYYY-MM-DD"))
		return nil, false
	}
	return &parsed, true
}
