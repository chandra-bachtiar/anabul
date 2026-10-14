package http

import (
	"net/http"

	"anabul/internal/agent"
)

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "agent belum aktif, coba lagi nanti",
		})
		return
	}
	var payload struct {
		Message string `json:"message"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	reply, err := s.chat.Chat(r.Context(), "web", payload.Message)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}

func (s *Server) handleConfirmAction(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent belum aktif"})
		return
	}
	reply, err := s.chat.Confirm(r.Context(), "web", r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}

func (s *Server) handleCancelAction(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent belum aktif"})
		return
	}
	reply, err := s.chat.Cancel(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}

var _ = agent.Reply{}
