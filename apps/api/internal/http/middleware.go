package http

import (
	"context"
	"net/http"
	"strings"

	"anabul/internal/alert"
)

// The app loads every asset from its own origin (fonts and mascots are
// embedded), so no external origin needs allowing. 'unsafe-inline' is limited
// to styles because Svelte injects component styles; inline scripts pass by hash
// instead.
const baseContentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'{SCRIPT_HASHES}; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"manifest-src 'self'; " +
	"base-uri 'none'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

// contentSecurityPolicy fills in the inline script hashes of the frontend build
// being served.
func (s *Server) contentSecurityPolicy() string {
	hashes := ""
	for _, hash := range s.scriptHashes {
		hashes += " " + hash
	}
	return strings.Replace(baseContentSecurityPolicy, "{SCRIPT_HASHES}", hashes, 1)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", s.contentSecurityPolicy())
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		header.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// statusRecorder keeps the status code for the alerting middleware.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}

// alertOnServerError notifies on 5xx responses. Health checks are excluded so
// monitoring does not turn into spam.
func (s *Server) alertOnServerError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.alert == nil || !s.alert.Enabled() {
			next.ServeHTTP(w, r)
			return
		}

		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		if recorder.status >= 500 && r.URL.Path != "/healthz" {
			s.alert.Send(context.WithoutCancel(r.Context()), alert.ServerError(r.Method, r.URL.Path, recorder.status))
		}
	})
}

// clientKey is the rate limit key. X-Forwarded-For is only trusted behind a
// reverse proxy (TRUST_PROXY), so a spoofed header cannot dodge the limit.
func (s *Server) clientKey(r *http.Request) string {
	if s.cfg.TrustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			first, _, _ := strings.Cut(forwarded, ",")
			if candidate := strings.TrimSpace(first); candidate != "" {
				return candidate
			}
		}
	}
	return remoteHost(r)
}

// secureRequest reports an HTTPS request, direct or through a trusted proxy.
func (s *Server) secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if s.cfg.TrustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return false
}
