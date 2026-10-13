package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/config"
	"anabul/migrations"
)

// cmd/migrate menjalankan seluruh migrasi lalu keluar, tanpa menyalakan
// server. Dipakai oleh `make migrate` dan saat deploy.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("gagal menyiapkan koneksi database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrations.Up(ctx, pool); err != nil {
		slog.Error("migrasi gagal", "error", err)
		os.Exit(1)
	}

	slog.Info("migrasi selesai")
}
