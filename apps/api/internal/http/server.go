package http

import (
	"context"
	"net/http"
	"time"

	"anabul/internal/agent"
	"anabul/internal/alert"
	"anabul/internal/auth"
	"anabul/internal/config"
	"anabul/internal/tools"
)

// Pinger is the slice of the database the HTTP layer needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ChatRunner is the agent behind the chat endpoints.
type ChatRunner interface {
	Chat(ctx context.Context, channel, message string) (agent.Reply, error)
	Confirm(ctx context.Context, channel, actionID string) (agent.Reply, error)
	Cancel(ctx context.Context, actionID string) (agent.Reply, error)
}

// Deps groups the server dependencies so callers do not have to remember the
// argument order.
type Deps struct {
	Cfg   config.Config
	DB    Pinger
	Web   http.Handler
	Tools *tools.Service
	Chat  ChatRunner
	Auth  *auth.Manager
	Alert *alert.Notifier
	// ScriptHashes are the CSP hashes for inline scripts in index.html.
	ScriptHashes []string
}

type Server struct {
	cfg          config.Config
	db           Pinger
	web          http.Handler
	tools        *tools.Service
	chat         ChatRunner
	auth         *auth.Manager
	alert        *alert.Notifier
	sessions     *sessionManager
	login        *rateLimiter
	scriptHashes []string
	nowFn        func() time.Time
}

func New(deps Deps) *Server {
	return &Server{
		cfg:          deps.Cfg,
		db:           deps.DB,
		web:          deps.Web,
		tools:        deps.Tools,
		chat:         deps.Chat,
		auth:         deps.Auth,
		alert:        deps.Alert,
		sessions:     newSessionManager(deps.Cfg.SessionSecret, deps.Cfg.SessionSecretPrevious),
		login:        newRateLimiter(10, 15*time.Minute),
		scriptHashes: deps.ScriptHashes,
		nowFn:        time.Now,
	}
}

func (s *Server) now() time.Time { return s.nowFn() }

// Handler returns the main router: /healthz, /api/*, the Telegram webhook, then
// the static frontend as the fallback.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("POST /api/auth/password", s.protect(s.handleChangePassword))
	mux.HandleFunc("POST /telegram/webhook", s.handleTelegramWebhook)

	api := func(method, pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+pattern, s.protect(handler))
	}

	api("GET", "/api/dashboard", s.handleDashboard)
	api("GET", "/api/summary", s.handleSummary)

	api("GET", "/api/transactions", s.handleListTransactions)
	api("POST", "/api/transactions", s.handleCreateTransaction)
	api("PATCH", "/api/transactions/{id}", s.handleUpdateTransaction)
	api("DELETE", "/api/transactions/{id}", s.handleDeleteTransaction)

	api("GET", "/api/categories", s.handleListCategories)
	api("POST", "/api/categories", s.handleCreateCategory)
	api("PATCH", "/api/categories/{id}", s.handleUpdateCategory)
	api("DELETE", "/api/categories/{id}", s.handleDeleteCategory)

	api("GET", "/api/accounts", s.handleListAccounts)
	api("POST", "/api/accounts", s.handleCreateAccount)

	api("GET", "/api/budgets", s.handleListBudgets)
	api("PUT", "/api/budgets", s.handleSetBudgets)
	api("POST", "/api/budgets/copy", s.handleCopyBudgets)

	api("GET", "/api/recurring-needs", s.handleListRecurringNeeds)
	api("POST", "/api/recurring-needs", s.handleCreateRecurringNeed)
	api("PATCH", "/api/recurring-needs/{id}", s.handleUpdateRecurringNeed)
	api("DELETE", "/api/recurring-needs/{id}", s.handleDeleteRecurringNeed)

	api("GET", "/api/needs", s.handleListNeeds)
	api("POST", "/api/needs/generate", s.handleGenerateNeeds)
	api("POST", "/api/needs/{id}/pay", s.handlePayNeed)
	api("POST", "/api/needs/{id}/unpay", s.handleUnpayNeed)

	api("GET", "/api/debts", s.handleListDebts)
	api("POST", "/api/debts", s.handleCreateDebt)
	api("GET", "/api/debts/{id}", s.handleGetDebt)
	api("POST", "/api/debts/{id}/payments", s.handleCreateDebtPayment)

	api("GET", "/api/months", s.handleListMonths)
	api("GET", "/api/months/{month}/preview", s.handleMonthPreview)
	api("POST", "/api/months/{month}/close", s.handleCloseMonth)
	api("POST", "/api/months/{month}/reopen", s.handleReopenMonth)

	api("POST", "/api/chat", s.handleChat)
	api("POST", "/api/actions/{id}/confirm", s.handleConfirmAction)
	api("POST", "/api/actions/{id}/cancel", s.handleCancelAction)

	mux.Handle("/", s.web)

	// Security headers sit outermost, then 5xx alerting.
	return s.securityHeaders(s.alertOnServerError(mux))
}

// protect lets only requests with a valid session through.
func (s *Server) protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authConfigured(r.Context()) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "login belum diaktifkan: isi APP_PASSWORD_HASH dan SESSION_SECRET di environment",
			})
			return
		}
		if !s.authenticated(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sesi tidak valid, login dulu ya"})
			return
		}
		s.refreshSession(w, r)
		next(w, r)
	}
}
