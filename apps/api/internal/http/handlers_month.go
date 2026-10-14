package http

import (
	"net/http"

	"anabul/internal/tools"
)

func (s *Server) handleListMonths(w http.ResponseWriter, r *http.Request) {
	locked, err := s.tools.ClosedMonths(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"closed_months": locked,
		"current_month": tools.FormatMonth(s.tools.CurrentMonth()),
	})
}

func (s *Server) handleMonthPreview(w http.ResponseWriter, r *http.Request) {
	preview, err := s.tools.MonthPreview(r.Context(), r.PathValue("month"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleCloseMonth(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Confirm bool `json:"confirm"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !payload.Confirm {
		writeError(w, tools.NewInvalidError("tutup bulan butuh konfirmasi: kirim confirm: true"))
		return
	}
	month := r.PathValue("month")
	snapshot, err := s.tools.CloseMonth(r.Context(), month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleReopenMonth(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Confirm bool `json:"confirm"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !payload.Confirm {
		writeError(w, tools.NewInvalidError("buka bulan butuh konfirmasi: kirim confirm: true"))
		return
	}
	month := r.PathValue("month")
	if err := s.tools.ReopenMonth(r.Context(), month); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "month": month, "reopened": true})
}
