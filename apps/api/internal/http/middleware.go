package http

import (
	"context"
	"net/http"
	"strings"

	"anabul/internal/alert"
)

// baseContentSecurityPolicy membatasi sumber daya ke origin sendiri. Frontend
// ANABUL memuat semua aset dari server yang sama (font dan maskot ikut
// di-embed), jadi tidak ada origin luar yang perlu diizinkan. 'unsafe-inline'
// hanya untuk style karena Svelte menyisipkan gaya komponen; skrip inline
// diizinkan lewat hash, bukan lewat unsafe-inline.
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

// contentSecurityPolicy menyusun header CSP lengkap dengan hash skrip inline
// dari build frontend yang sedang dilayani.
func (s *Server) contentSecurityPolicy() string {
	hashes := ""
	for _, hash := range s.scriptHashes {
		hashes += " " + hash
	}
	return strings.Replace(baseContentSecurityPolicy, "{SCRIPT_HASHES}", hashes, 1)
}

// securityHeaders menambahkan header pengaman ke semua respons.
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

// statusRecorder mencatat kode status untuk kebutuhan alert.
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

// alertOnServerError mengirim notifikasi saat ada respons 5xx. Health check
// dikecualikan supaya monitoring tidak menghasilkan spam.
func (s *Server) alertOnServerError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.alert == nil || !s.alert.Enabled() {
			next.ServeHTTP(w, r)
			return
		}

		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		if recorder.status >= 500 && r.URL.Path != "/healthz" {
			s.alert.Kirim(context.WithoutCancel(r.Context()), alert.PesanServer(r.Method, r.URL.Path, recorder.status))
		}
	})
}

// clientKey menentukan kunci pembatas percobaan login. Header X-Forwarded-For
// hanya dipercaya kalau server memang berada di belakang reverse proxy
// (TRUST_PROXY), supaya tidak bisa dipalsukan untuk melewati rate limit.
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

// secureRequest menentukan apakah permintaan datang lewat HTTPS, langsung atau
// lewat proxy yang dipercaya.
func (s *Server) secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if s.cfg.TrustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return false
}
