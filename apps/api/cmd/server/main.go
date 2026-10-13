package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/config"
	httpapi "anabul/internal/http"
	"anabul/internal/webui"
	"anabul/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("server berhenti dengan error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.Up(ctx, pool); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           httpapi.New(cfg, pool, webui.Handler()).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("anabul siap menerima request", "addr", srv.Addr, "timezone", cfg.Timezone)
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
// baru dinyalakan).
func connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
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
