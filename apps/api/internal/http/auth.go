package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie  = "anabul_session"
	sessionTTL     = 30 * 24 * time.Hour
	sessionRefresh = 7 * 24 * time.Hour
)

var errUnauthorized = errors.New("sesi tidak valid")

// sessionManager menandatangani cookie sesi dengan HMAC-SHA256. Secret lama
// tetap diterima untuk verifikasi supaya rotasi SESSION_SECRET tidak langsung
// memaksa semua orang login ulang.
type sessionManager struct {
	secret     []byte
	sebelumnya []byte
}

func newSessionManager(secret, sebelumnya string) *sessionManager {
	return &sessionManager{
		secret:     []byte(secret),
		sebelumnya: []byte(strings.TrimSpace(sebelumnya)),
	}
}

func signWith(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *sessionManager) sign(payload string) string {
	return signWith(m.secret, payload)
}

func (m *sessionManager) issue(now time.Time) string {
	payload := "exp=" + strconv.FormatInt(now.Add(sessionTTL).Unix(), 10)
	return payload + "." + m.sign(payload)
}

func (m *sessionManager) verify(value string, now time.Time) (time.Time, error) {
	payload, signature, found := strings.Cut(value, ".")
	if !found {
		return time.Time{}, errUnauthorized
	}

	cocok := hmac.Equal([]byte(signature), []byte(m.sign(payload)))
	if !cocok && len(m.sebelumnya) > 0 {
		cocok = hmac.Equal([]byte(signature), []byte(signWith(m.sebelumnya, payload)))
	}
	if !cocok {
		return time.Time{}, errUnauthorized
	}

	expires := strings.TrimPrefix(payload, "exp=")
	unix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return time.Time{}, errUnauthorized
	}
	deadline := time.Unix(unix, 0)
	if now.After(deadline) {
		return time.Time{}, errUnauthorized
	}
	return deadline, nil
}

func (s *Server) authenticated(r *http.Request) bool {
	if !s.authConfigured(r.Context()) {
		return false
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	_, err = s.sessions.verify(cookie.Value, s.now())
	return err == nil
}

// authConfigured menandai password sudah diatur (di database atau environment).
func (s *Server) authConfigured(ctx context.Context) bool {
	if s.cfg.SessionSecret == "" || s.auth == nil {
		return false
	}
	configured, err := s.auth.Configured(ctx)
	if err != nil {
		slog.Error("gagal memeriksa konfigurasi password", "error", err)
		return false
	}
	return configured
}

// setSessionCookie mengirim cookie sesi baru. Atribut Secure dimatikan saat
// diakses lewat HTTP lokal supaya tetap bisa dipakai saat development.
func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, now time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.sessions.issue(now),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// rateLimiter membatasi percobaan login per alamat IP.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{attempts: make(map[string][]time.Time), limit: limit, window: window}
}

func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	kept := make([]time.Time, 0, len(l.attempts[key]))
	for _, attempt := range l.attempts[key] {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

func (l *rateLimiter) forget(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// remoteHost mengambil alamat klien tanpa port.
func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
