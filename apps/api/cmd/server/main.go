package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/agent"
	"anabul/internal/alert"
	"anabul/internal/auth"
	"anabul/internal/config"
	httpapi "anabul/internal/http"
	"anabul/internal/sched"
	"anabul/internal/store"
	"anabul/internal/tools"
	"anabul/internal/webui"
	"anabul/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("server berhenti dengan error", "error", err)
		laporErrorFatal(err)
		os.Exit(1)
	}
}

// laporErrorFatal mengirim notifikasi saat server gagal start. Konfigurasi
// dibaca langsung dari environment karena run() bisa berhenti sebelum
// notifier dibuat.
func laporErrorFatal(err error) {
	url := strings.TrimSpace(os.Getenv("ALERT_WEBHOOK_URL"))
	if url == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	alert.New(url).Kirim(ctx, "server berhenti: "+err.Error())
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("zona waktu %q tidak dikenal: %w", cfg.Timezone, err)
	}

	pool, err := connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.Up(ctx, pool); err != nil {
		return err
	}

	queries := store.New(pool)
	service := tools.New(queries, pool, location)

	alerts := alert.New(cfg.AlertWebhookURL)
	if alerts.Enabled() {
		slog.Info("notifikasi error aktif", "via", "ALERT_WEBHOOK_URL")
	}

	authManager := auth.New(queries, cfg.AppPasswordHash)
	seeded, err := authManager.EnsureSeeded(ctx)
	if err != nil {
		return fmt.Errorf("siapkan password: %w", err)
	}
	if seeded {
		slog.Info("password awal disalin dari environment ke database, bisa diganti dari Pengaturan")
	}

	chatbot := agent.New(agent.Config{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
	}, service, queries)
	if !chatbot.LLMEnabled() {
		slog.Warn("LLM_API_KEY belum diisi, agent memakai parser sederhana")
	}
	if configured, err := authManager.Configured(ctx); err == nil && !configured {
		slog.Warn("APP_PASSWORD_HASH belum diisi, endpoint /api tidak bisa dipakai")
	} else if err == nil {
		if usingDefault, err := authManager.UsingDefault(ctx); err == nil && usingDefault {
			slog.Warn("password masih memakai default, sebaiknya diganti dari halaman Pengaturan")
		}
	}

	scheduler := sched.New(service)
	if err := scheduler.Start(ctx); err != nil {
		return fmt.Errorf("nyalakan scheduler: %w", err)
	}
	defer scheduler.Stop()

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Handler: httpapi.New(httpapi.Deps{
			Cfg:   cfg,
			DB:    pool,
			Web:   webui.Handler(),
			Tools: service,
			Chat:  chatbot,
			Auth:  authManager,
			Alert: alerts,
			// Skrip inline dari SvelteKit diizinkan lewat hash, bukan unsafe-inline.
			ScriptHashes: webui.Hashes(),
		}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("anabul siap menerima request", "addr", srv.Addr, "timezone", cfg.Timezone, "llm", chatbot.LLMEnabled())
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	slog.Info("menerima sinyal berhenti, menutup server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// connect membuka pool dan menunggu database siap (misal saat docker compose
// baru dinyalakan, atau Neon yang baru bangun dari idle).
func connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	// Endpoint pooler Neon (PgBouncer) tidak mendukung prepared statement
	// lintas transaksi, jadi query dikirim langsung.
	if strings.Contains(cfg.ConnConfig.Host, "-pooler") {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Jakarta"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("siapkan pool database: %w", err)
	}

	const attempts = 10
	var pingErr error
	for i := 1; i <= attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr = pool.Ping(pingCtx)
		cancel()
		if pingErr == nil {
			return pool, nil
		}
		slog.Warn("menunggu database siap", "percobaan", i, "error", pingErr)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("database tidak dapat dihubungi: %w", pingErr)
}
