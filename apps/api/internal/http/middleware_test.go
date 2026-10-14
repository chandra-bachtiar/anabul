package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"anabul/internal/alert"
	"anabul/internal/config"
)

func TestSecurityHeaders(t *testing.T) {
	server := New(Deps{
		Cfg:          config.Config{Timezone: "Asia/Jakarta"},
		ScriptHashes: []string{"'sha256-abc123'"},
	})

	handler := server.securityHeaders(server.alertOnServerError(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	header := recorder.Header()
	for nama, ingin := range map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"Referrer-Policy":            "no-referrer",
		"X-Frame-Options":            "DENY",
		"Cross-Origin-Opener-Policy": "same-origin",
	} {
		if got := header.Get(nama); got != ingin {
			t.Fatalf("header %s = %q, ingin %q", nama, got, ingin)
		}
	}

	csp := header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("CSP harus terpasang")
	}
	if !strings.Contains(csp, "script-src 'self' 'sha256-abc123'") {
		t.Fatalf("CSP harus memuat hash skrip inline, dapat %q", csp)
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatal("script-src tidak boleh memakai unsafe-inline")
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("CSP harus melarang iframe, dapat %q", csp)
	}
}

func TestAlertHanyaUntukErrorServer(t *testing.T) {
	var terkirim []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		terkirim = append(terkirim, r.URL.Path)
	}))
	defer server.Close()

	// Notifier async; panggil langsung lewat s.alert supaya bisa diperiksa.
	serverHTTP := New(Deps{Cfg: config.Config{}, Alert: alert.New(server.URL)})

	cases := []struct {
		nama       string
		status     int
		path       string
		harusKirim bool
	}{
		{nama: "5xx dikirim", status: http.StatusInternalServerError, path: "/api/transactions", harusKirim: true},
		{nama: "4xx tidak dikirim", status: http.StatusBadRequest, path: "/api/transactions", harusKirim: false},
		{nama: "healthz dikecualikan", status: http.StatusInternalServerError, path: "/healthz", harusKirim: false},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			terkirim = nil
			handler := serverHTTP.alertOnServerError(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))

			// Kirim() berjalan sinkron, jadi hasilnya sudah bisa dibaca.
			if tc.harusKirim && len(terkirim) == 0 {
				t.Fatal("alert harus dikirim untuk error 5xx")
			}
			if !tc.harusKirim && len(terkirim) > 0 {
				t.Fatalf("alert tidak boleh dikirim, tapi terkirim %d kali", len(terkirim))
			}
		})
	}
}

func TestClientKeyHormatiTrustProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.RemoteAddr = "10.0.0.9:5555"
	request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	tanpaProxy := New(Deps{Cfg: config.Config{}})
	if got := tanpaProxy.clientKey(request); got != "10.0.0.9" {
		t.Fatalf("tanpa TRUST_PROXY = %q, ingin alamat koneksi langsung", got)
	}

	denganProxy := New(Deps{Cfg: config.Config{TrustProxy: true}})
	if got := denganProxy.clientKey(request); got != "203.0.113.7" {
		t.Fatalf("dengan TRUST_PROXY = %q, ingin alamat asli dari header", got)
	}
}

func TestSecureRequestIkutTrustProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")

	tanpaProxy := New(Deps{Cfg: config.Config{}})
	if tanpaProxy.secureRequest(request) {
		t.Fatal("tanpa TRUST_PROXY, header X-Forwarded-Proto tidak boleh dipercaya")
	}

	denganProxy := New(Deps{Cfg: config.Config{TrustProxy: true}})
	if !denganProxy.secureRequest(request) {
		t.Fatal("dengan TRUST_PROXY, permintaan HTTPS lewat proxy harus dikenali")
	}
}

func TestRotasiSessionSecret(t *testing.T) {
	now := time.Now()
	lama := newSessionManager("secret-lama", "")
	cookieLama := lama.issue(now)

	baru := newSessionManager("secret-baru", "secret-lama")
	if _, err := baru.verify(cookieLama, now); err != nil {
		t.Fatalf("cookie dari secret lama harus tetap diterima saat rotasi: %v", err)
	}

	cookieBaru := baru.issue(now)
	if _, err := baru.verify(cookieBaru, now); err != nil {
		t.Fatalf("cookie baru harus valid: %v", err)
	}

	// Cookie baru tidak boleh valid di server lama.
	if _, err := lama.verify(cookieBaru, now); err == nil {
		t.Fatal("cookie bertanda tangan secret baru tidak boleh lolos di secret lama")
	}

	// Tanpa secret sebelumnya, cookie lama ditolak.
	tanpa := newSessionManager("secret-baru", "")
	if _, err := tanpa.verify(cookieLama, now); err == nil {
		t.Fatal("tanpa SESSION_SECRET_PREVIOUS, cookie lama harus ditolak")
	}
}
