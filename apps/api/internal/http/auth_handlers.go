package http

import (
	"net/http"
	"time"

	"anabul/internal/auth"
)

type loginRequest struct {
	Password string `json:"password"`
}

const passwordHint = "minimal 8 karakter, dan jangan pakai password default"

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authConfigured(r.Context()) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "login belum diaktifkan: isi APP_PASSWORD_HASH dan SESSION_SECRET di environment",
		})
		return
	}

	key := s.clientKey(r)
	now := s.now()
	if !s.login.allow(key, now) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "kebanyakan percobaan login, tunggu beberapa menit ya",
		})
		return
	}

	var payload loginRequest
	if !decodeJSON(w, r, &payload) {
		return
	}

	cocok, err := s.auth.Verify(r.Context(), payload.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	if !cocok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "password salah"})
		return
	}

	s.login.forget(key)
	s.setSessionCookie(w, r, now)

	usingDefault, err := s.auth.UsingDefault(r.Context())
	if err != nil {
		usingDefault = false
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"expires_at":             now.Add(sessionTTL),
		"using_default_password": usingDefault,
		"password_hint":          passwordHint,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleMe dipakai frontend untuk memeriksa sesi dan menyarankan ganti password.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if !s.authenticated(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"configured":    s.authConfigured(r.Context()),
		})
		return
	}

	usingDefault, err := s.auth.UsingDefault(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	s.refreshSession(w, r)
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated":          true,
		"configured":             true,
		"using_default_password": usingDefault,
	})
}

// handleChangePassword mengganti password aplikasi dari dalam app.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	key := s.clientKey(r)
	now := s.now()
	if !s.login.allow(key, now) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "kebanyakan percobaan, tunggu beberapa menit ya",
		})
		return
	}

	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := s.auth.Set(r.Context(), payload.CurrentPassword, payload.NewPassword); err != nil {
		writeError(w, err)
		return
	}

	s.login.forget(key)
	s.setSessionCookie(w, r, now)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"using_default_password": false,
		"password_hint":          passwordHint,
		"min_length":             auth.MinPasswordLength,
	})
}

// refreshSession memperpanjang cookie kalau umurnya hampir habis.
func (s *Server) refreshSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return
	}
	now := s.now()
	expires, err := s.sessions.verify(cookie.Value, now)
	if err != nil {
		return
	}
	if time.Until(expires) < sessionRefresh {
		s.setSessionCookie(w, r, now)
	}
}
