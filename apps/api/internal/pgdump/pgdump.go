// Package pgdump menjalankan pg_dump dan psql lewat container Postgres resmi.
// Dipakai cmd/backup dan cmd/restore supaya keduanya jalan sama di Windows,
// macOS, dan Linux tanpa memasang klien Postgres di mesin.
package pgdump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const image = "postgres:16-alpine"

// Dump menulis hasil pg_dump (SQL teks biasa) ke dst.
func Dump(ctx context.Context, databaseURL string, dst io.Writer) error {
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i",
		"--add-host", "host.docker.internal:host-gateway",
		"-e", "PGURL="+siapkanURL(databaseURL),
		image, "sh", "-c", `pg_dump --no-owner --no-privileges "$PGURL"`)
	cmd.Stdout = dst

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump gagal: %w (%s)", err, ringkas(stderr.String()))
	}
	return nil
}

// Restore mengirim isi SQL dari src ke psql.
func Restore(ctx context.Context, databaseURL string, src io.Reader) error {
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i",
		"--add-host", "host.docker.internal:host-gateway",
		"-e", "PGURL="+siapkanURL(databaseURL),
		image, "sh", "-c", `psql --quiet --set ON_ERROR_STOP=1 "$PGURL"`)
	cmd.Stdin = src

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql gagal: %w (%s)", err, ringkas(stderr.String()))
	}
	return nil
}

// siapkanURL mengganti host lokal menjadi host.docker.internal karena pg_dump
// dan psql dijalankan dari dalam container, bukan di mesin ini.
func siapkanURL(databaseURL string) string {
	for _, host := range []string{"@localhost", "@127.0.0.1"} {
		if strings.Contains(databaseURL, host) {
			return strings.Replace(databaseURL, host, "@host.docker.internal", 1)
		}
	}
	return databaseURL
}

// Tersedia memeriksa docker ada di PATH dan daemonnya hidup.
func Tersedia(ctx context.Context) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker tidak ditemukan di PATH: %w", err)
	}
	if err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").Run(); err != nil {
		return fmt.Errorf("daemon docker belum siap: %w", err)
	}
	return nil
}

func ringkas(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 300 {
		return text[:300] + "..."
	}
	return text
}
