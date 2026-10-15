// Package pgdump runs pg_dump and psql via the official Postgres container, so
// cmd/backup and cmd/restore work the same everywhere without a local client.
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

// Dump writes the pg_dump output (plain-text SQL) to dst.
func Dump(ctx context.Context, databaseURL string, dst io.Writer) error {
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i",
		"--add-host", "host.docker.internal:host-gateway",
		"-e", "PGURL="+prepareURL(databaseURL),
		image, "sh", "-c", `pg_dump --no-owner --no-privileges "$PGURL"`)
	cmd.Stdout = dst

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w (%s)", err, shorten(stderr.String()))
	}
	return nil
}

// Restore feeds the SQL stream from src into psql.
func Restore(ctx context.Context, databaseURL string, src io.Reader) error {
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i",
		"--add-host", "host.docker.internal:host-gateway",
		"-e", "PGURL="+prepareURL(databaseURL),
		image, "sh", "-c", `psql --quiet --set ON_ERROR_STOP=1 "$PGURL"`)
	cmd.Stdin = src

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql failed: %w (%s)", err, shorten(stderr.String()))
	}
	return nil
}

// prepareURL rewrites a local host to host.docker.internal because pg_dump and
// psql run inside the container, not on this machine.
func prepareURL(databaseURL string) string {
	for _, host := range []string{"@localhost", "@127.0.0.1"} {
		if strings.Contains(databaseURL, host) {
			return strings.Replace(databaseURL, host, "@host.docker.internal", 1)
		}
	}
	return databaseURL
}

// Available checks that docker is on PATH and its daemon is running.
func Available(ctx context.Context) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH: %w", err)
	}
	if err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").Run(); err != nil {
		return fmt.Errorf("docker daemon is not ready: %w", err)
	}
	return nil
}

func shorten(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 300 {
		return text[:300] + "..."
	}
	return text
}
