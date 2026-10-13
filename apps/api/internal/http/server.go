package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"anabul/internal/config"
)

// Pinger adalah bagian database yang dibutuhkan layer HTTP.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server merangkai seluruh route aplikasi.
type Server struct {
	cfg config.Config
	db  Pinger
	web http.Handler
}

func New(cfg config.Config, db Pinger, web http.Handler) *Server {
	return &Server{cfg: cfg, db: db, web: web}
}

// Handler mengembalikan router utama: /healthz, /api/* (menyusul di fase
// berikutnya), lalu frontend statis sebagai fallback.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.Handle("/", s.web)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("gagal menulis respons json", "error", err)
	}
}
