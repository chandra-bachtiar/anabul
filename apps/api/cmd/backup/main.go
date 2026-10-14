// cmd/backup menyimpan dump database ke folder backups/ lalu menghapus
// backup lama sesuai BACKUP_KEEP.
//
// Contoh:
//
//	go run ./cmd/backup
//	BACKUP_DIR=D:\backup-anabul BACKUP_KEEP=30 go run ./cmd/backup
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
	if err := jalankan(); err != nil {
		fmt.Fprintln(os.Stderr, "backup gagal:", err)
		os.Exit(1)
	}
}

func jalankan() error {
	if err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("baca .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := pgdump.Tersedia(ctx); err != nil {
		return err
	}

	dir := env("BACKUP_DIR", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("siapkan folder %s: %w", dir, err)
	}

	nama := fmt.Sprintf("anabul-%s.sql.gz", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, nama)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("buat berkas backup: %w", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	if err := pgdump.Dump(ctx, cfg.DatabaseURL, gz); err != nil {
		file.Close()
		// Jangan tinggalkan berkas setengah jadi supaya tidak dikira backup sah.
		_ = os.Remove(path)
		return err
	}
	if err := gz.Close(); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("tutup gzip: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("tutup berkas: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		_ = os.Remove(path)
		return fmt.Errorf("berkas backup kosong: %s", path)
	}

	dihapus, err := rapikan(dir, keep())
	if err != nil {
		return err
	}

	fmt.Printf("backup tersimpan: %s (%.1f KB)\n", path, float64(info.Size())/1024)
	if dihapus > 0 {
		fmt.Printf("backup lama dihapus: %d (sisakan %d terbaru)\n", dihapus, keep())
	}
	return nil
}

func keep() int {
	value, err := strconv.Atoi(env("BACKUP_KEEP", "14"))
	if err != nil || value < 1 {
		return 14
	}
	return value
}

// rapikan menyisakan sejumlah berkas terbaru dan menghapus sisanya.
func rapikan(dir string, simpan int) (int, error) {
	berkas, err := filepath.Glob(filepath.Join(dir, "anabul-*.sql.gz"))
	if err != nil {
		return 0, err
	}
	if len(berkas) <= simpan {
		return 0, nil
	}

	sort.Slice(berkas, func(i, j int) bool {
		infoI, errI := os.Stat(berkas[i])
		infoJ, errJ := os.Stat(berkas[j])
		if errI != nil || errJ != nil {
			return berkas[i] > berkas[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	dihapus := 0
	for _, path := range berkas[simpan:] {
		if err := os.Remove(path); err != nil {
			return dihapus, fmt.Errorf("hapus %s: %w", path, err)
		}
		dihapus++
	}
	return dihapus, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
