// cmd/backup dumps the database into backups/ and prunes old dumps down to the
// newest BACKUP_KEEP files, e.g. BACKUP_DIR=D:\backup-anabul go run ./cmd/backup.
package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"anabul/internal/config"
	"anabul/internal/pgdump"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "backup failed:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := pgdump.Available(ctx); err != nil {
		return err
	}

	dir := env("BACKUP_DIR", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare folder %s: %w", dir, err)
	}

	name := fmt.Sprintf("anabul-%s.sql.gz", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	if err := pgdump.Dump(ctx, cfg.DatabaseURL, gz); err != nil {
		file.Close()
		// Never leave a half-written file behind: it could pass for a valid backup.
		_ = os.Remove(path)
		return err
	}
	if err := gz.Close(); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		_ = os.Remove(path)
		return fmt.Errorf("backup file is empty: %s", path)
	}

	removed, err := prune(dir, keepCount())
	if err != nil {
		return err
	}

	fmt.Printf("backup saved: %s (%.1f KB)\n", path, float64(info.Size())/1024)
	if removed > 0 {
		fmt.Printf("old backups removed: %d (kept %d newest)\n", removed, keepCount())
	}
	return nil
}

func keepCount() int {
	value, err := strconv.Atoi(env("BACKUP_KEEP", "14"))
	if err != nil || value < 1 {
		return 14
	}
	return value
}

// prune keeps the newest keepCount files and removes the rest.
func prune(dir string, keepCount int) (int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "anabul-*.sql.gz"))
	if err != nil {
		return 0, err
	}
	if len(files) <= keepCount {
		return 0, nil
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] > files[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	removed := 0
	for _, path := range files[keepCount:] {
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("remove %s: %w", path, err)
		}
		removed++
	}
	return removed, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
