// cmd/restore loads a dump produced by cmd/backup into the database named by
// DATABASE_URL, overwriting existing rows; usage: go run ./cmd/restore <backup.sql.gz>.
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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "restore failed:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: go run ./cmd/restore <backup-file.sql.gz>")
	}
	path := strings.TrimSpace(os.Args[1])

	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()

	var src io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("read gzip: %w", err)
		}
		defer gz.Close()
		src = gz
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := pgdump.Available(ctx); err != nil {
		return err
	}
	if err := pgdump.Restore(ctx, cfg.DatabaseURL, src); err != nil {
		return err
	}

	fmt.Printf("restore complete from %s\n", path)
	return nil
}
