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
		slog.Error("server exited with error", "error", err)
		reportFatalError(err)
		os.Exit(1)
	}
}

// reportFatalError sends an alert when the server fails to start. Config is read
// straight from the environment because run() can fail before the notifier exists.
func reportFatalError(err error) {
	url := strings.TrimSpace(os.Getenv("ALERT_WEBHOOK_URL"))
	if url == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	alert.New(url).Send(ctx, "server stopped: "+err.Error())
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
		return fmt.Errorf("unknown timezone %q: %w", cfg.Timezone, err)
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
		slog.Info("error notifications enabled", "via", "ALERT_WEBHOOK_URL")
	}

	authManager := auth.New(queries, cfg.AppPasswordHash)
	seeded, err := authManager.EnsureSeeded(ctx)
	if err != nil {
		return fmt.Errorf("prepare password: %w", err)
	}
	if seeded {
		slog.Info("initial password copied from the environment to the database, it can be changed from the app settings")
	}

	chatbot := agent.New(agent.Config{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
	}, service, queries)
	if !chatbot.LLMEnabled() {
		slog.Warn("LLM_API_KEY is not set, the agent uses the simple parser")
	}
	if configured, err := authManager.Configured(ctx); err == nil && !configured {
		slog.Warn("APP_PASSWORD_HASH is not set, the /api endpoints are unavailable")
	} else if err == nil {
		if usingDefault, err := authManager.UsingDefault(ctx); err == nil && usingDefault {
			slog.Warn("password is still the default, change it from the app settings")
		}
	}

	scheduler := sched.New(service)
	if err := scheduler.Start(ctx); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
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
			// SvelteKit inline scripts are allowed via hashes, not unsafe-inline.
			ScriptHashes: webui.Hashes(),
		}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("anabul ready to serve requests", "addr", srv.Addr, "timezone", cfg.Timezone, "llm", chatbot.LLMEnabled())
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

	slog.Info("shutdown signal received, closing server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// connect opens the pool and waits for the database to come up, e.g. right after
// docker compose starts or when a Neon instance wakes from idle.
func connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	// Neon's pooler (PgBouncer) has no cross-transaction prepared statements, so
	// queries are sent directly.
	if strings.Contains(cfg.ConnConfig.Host, "-pooler") {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Jakarta"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
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
		slog.Warn("waiting for database", "attempt", i, "error", pingErr)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("database unreachable: %w", pingErr)
}
