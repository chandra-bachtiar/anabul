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
	for name, want := range map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"Referrer-Policy":            "no-referrer",
		"X-Frame-Options":            "DENY",
		"Cross-Origin-Opener-Policy": "same-origin",
	} {
		if got := header.Get(name); got != want {
			t.Fatalf("header %s = %q, want %q", name, got, want)
		}
	}

	csp := header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("the CSP header must be set")
	}
	if !strings.Contains(csp, "script-src 'self' 'sha256-abc123'") {
		t.Fatalf("the CSP must carry the inline script hash, got %q", csp)
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatal("script-src must not use unsafe-inline")
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("the CSP must forbid iframes, got %q", csp)
	}
}

func TestAlertOnlyForServerErrors(t *testing.T) {
	var sent []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = append(sent, r.URL.Path)
	}))
	defer server.Close()

	// The notifier posts asynchronously; going through s.alert keeps the check
	// synchronous.
	serverHTTP := New(Deps{Cfg: config.Config{}, Alert: alert.New(server.URL)})

	cases := []struct {
		name       string
		status     int
		path       string
		shouldSend bool
	}{
		{name: "5xx is sent", status: http.StatusInternalServerError, path: "/api/transactions", shouldSend: true},
		{name: "4xx is not sent", status: http.StatusBadRequest, path: "/api/transactions", shouldSend: false},
		{name: "healthz is excluded", status: http.StatusInternalServerError, path: "/healthz", shouldSend: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sent = nil
			handler := serverHTTP.alertOnServerError(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if tc.shouldSend && len(sent) == 0 {
				t.Fatal("a 5xx must trigger an alert")
			}
			if !tc.shouldSend && len(sent) > 0 {
				t.Fatalf("no alert expected, but %d were sent", len(sent))
			}
		})
	}
}

func TestClientKeyRespectsTrustProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.RemoteAddr = "10.0.0.9:5555"
	request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	withoutProxy := New(Deps{Cfg: config.Config{}})
	if got := withoutProxy.clientKey(request); got != "10.0.0.9" {
		t.Fatalf("without TRUST_PROXY = %q, want the connection address", got)
	}

	withProxy := New(Deps{Cfg: config.Config{TrustProxy: true}})
	if got := withProxy.clientKey(request); got != "203.0.113.7" {
		t.Fatalf("with TRUST_PROXY = %q, want the forwarded address", got)
	}
}

func TestSecureRequestFollowsTrustProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")

	withoutProxy := New(Deps{Cfg: config.Config{}})
	if withoutProxy.secureRequest(request) {
		t.Fatal("without TRUST_PROXY, X-Forwarded-Proto must not be trusted")
	}

	withProxy := New(Deps{Cfg: config.Config{TrustProxy: true}})
	if !withProxy.secureRequest(request) {
		t.Fatal("with TRUST_PROXY, a proxied HTTPS request must be recognised")
	}
}

func TestSessionSecretRotation(t *testing.T) {
	now := time.Now()
	old := newSessionManager("old-secret", "")
	oldCookie := old.issue(now)

	fresh := newSessionManager("new-secret", "old-secret")
	if _, err := fresh.verify(oldCookie, now); err != nil {
		t.Fatalf("a cookie from the old secret must still verify during rotation: %v", err)
	}

	newCookie := fresh.issue(now)
	if _, err := fresh.verify(newCookie, now); err != nil {
		t.Fatalf("a fresh cookie must verify: %v", err)
	}

	// Cookies signed with the new secret stay invalid on the old server.
	if _, err := old.verify(newCookie, now); err == nil {
		t.Fatal("a cookie signed with the new secret must not pass the old one")
	}

	// Without the previous secret, old cookies are rejected.
	withoutPrevious := newSessionManager("new-secret", "")
	if _, err := withoutPrevious.verify(oldCookie, now); err == nil {
		t.Fatal("without SESSION_SECRET_PREVIOUS, old cookies must be rejected")
	}
}
