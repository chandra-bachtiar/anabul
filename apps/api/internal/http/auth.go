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

var errUnauthorized = errors.New("invalid session")

// sessionManager signs the session cookie with HMAC-SHA256. The previous secret
// stays valid for verification, so rotating SESSION_SECRET does not log everyone out.
type sessionManager struct {
	secret   []byte
	previous []byte
}

func newSessionManager(secret, previous string) *sessionManager {
	return &sessionManager{
		secret:   []byte(secret),
		previous: []byte(strings.TrimSpace(previous)),
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

	match := hmac.Equal([]byte(signature), []byte(m.sign(payload)))
	if !match && len(m.previous) > 0 {
		match = hmac.Equal([]byte(signature), []byte(signWith(m.previous, payload)))
	}
	if !match {
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

// authConfigured reports whether a password exists in the database or environment.
func (s *Server) authConfigured(ctx context.Context) bool {
	if s.cfg.SessionSecret == "" || s.auth == nil {
		return false
	}
	configured, err := s.auth.Configured(ctx)
	if err != nil {
		slog.Error("failed to check the password configuration", "error", err)
		return false
	}
	return configured
}

// setSessionCookie drops Secure on plain local HTTP so development keeps working.
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

// rateLimiter caps login attempts per address.
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

// remoteHost strips the port from the client address.
func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
