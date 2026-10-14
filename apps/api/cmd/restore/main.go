// cmd/restore mengembalikan dump dari cmd/backup ke database tujuan.
//
// Contoh:
//
//	go run ./cmd/restore backups/anabul-20261014-070000.sql.gz
//	DATABASE_URL=postgres://... go run ./cmd/restore backup.sql.gz
//
// Awas: restore menimpa isi tabel yang ada, jadi pastikan DATABASE_URL menunjuk
// ke database yang benar (mis. database baru) sebelum menjalankannya.
package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"anabul/internal/config"
	"anabul/internal/pgdump"
)

func main() {
	if err := jalankan(); err != nil {
		fmt.Fprintln(os.Stderr, "restore gagal:", err)
		os.Exit(1)
	}
}

func jalankan() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("pakai: go run ./cmd/restore <berkas-backup.sql.gz>")
	}
	path := strings.TrimSpace(os.Args[1])

	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("baca .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("buka berkas backup: %w", err)
	}
	defer file.Close()

	var src io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("baca gzip: %w", err)
		}
		defer gz.Close()
		src = gz
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := pgdump.Tersedia(ctx); err != nil {
		return err
	}
	if err := pgdump.Restore(ctx, cfg.DatabaseURL, src); err != nil {
		return err
	}

	fmt.Printf("restore selesai dari %s\n", path)
	return nil
}
