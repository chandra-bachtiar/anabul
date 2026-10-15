package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/store"
	"anabul/migrations"
)

func TestValidateNewPassword(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		next     string
		wantErr  bool
		contains string
	}{
		{name: "valid", current: "anabul123", next: "kopi-susu-2026", wantErr: false},
		{name: "blank", current: "anabul123", next: "   ", wantErr: true, contains: "belum diisi"},
		{name: "too short", current: "anabul123", next: "pendek", wantErr: true, contains: "minimal"},
		{name: "same as the current one", current: "rahasia-lama", next: "rahasia-lama", wantErr: true, contains: "masih sama"},
		{name: "still the default password", current: "rahasia-lama", next: DefaultPassword, wantErr: true, contains: "default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNewPassword(tc.current, tc.next)
			if tc.wantErr && err == nil {
				t.Fatal("wanted an error, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("wanted no error, got: %v", err)
			}
			if tc.contains != "" && (err == nil || !strings.Contains(err.Error(), tc.contains)) {
				t.Fatalf("error %v does not contain %q", err, tc.contains)
			}
		})
	}
}

// TestManagerChangePassword needs a test database; it is skipped without
// ANABUL_TEST_DATABASE_URL so `make test` still runs offline.
func TestManagerChangePassword(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ANABUL_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ANABUL_TEST_DATABASE_URL is empty, skipping the password change test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to the test database: %v", err)
	}
	// t.Cleanup runs LIFO: registering the pool close first makes the data
	// cleanup run while the connection is still open.
	t.Cleanup(pool.Close)

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrate the test database: %v", err)
	}

	// Start clean and restore the state afterwards, so the app falls back to the
	// environment hash.
	if _, err := pool.Exec(ctx, "DELETE FROM app_settings WHERE key = $1", settingKey); err != nil {
		t.Fatalf("clear app_settings: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM app_settings WHERE key = $1", settingKey); err != nil {
			t.Errorf("failed to clear app_settings: %v", err)
		}
	})

	queries := store.New(pool)
	envHash := "$2a$10$tJTejoXklz5EivrCcRa/DuJc/ojCxtGB.PWzaWeGLSapcZkNbS3ga" // bcrypt hash of "anabul123"

	manager := New(queries, envHash)
	seeded, err := manager.EnsureSeeded(ctx)
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}
	if !seeded {
		t.Fatal("the environment hash must be copied into the database on first start")
	}

	configured, err := manager.Configured(ctx)
	if err != nil || !configured {
		t.Fatalf("Configured = %v (err %v), want true", configured, err)
	}

	usingDefault, err := manager.UsingDefault(ctx)
	if err != nil || !usingDefault {
		t.Fatalf("UsingDefault = %v (err %v), want true before the change", usingDefault, err)
	}

	if ok, _ := manager.Verify(ctx, "salah"); ok {
		t.Fatal("a wrong password must not pass")
	}
	if ok, _ := manager.Verify(ctx, DefaultPassword); !ok {
		t.Fatal("the default password must pass before it is changed")
	}

	if err := manager.Set(ctx, DefaultPassword, "kopi-susu-2026"); err != nil {
		t.Fatalf("change the password: %v", err)
	}

	if ok, _ := manager.Verify(ctx, DefaultPassword); ok {
		t.Fatal("the old password must stop working")
	}
	if ok, _ := manager.Verify(ctx, "kopi-susu-2026"); !ok {
		t.Fatal("the new password must pass")
	}
	if usingDefault, _ := manager.UsingDefault(ctx); usingDefault {
		t.Fatal("UsingDefault must be false after the change")
	}

	// A fresh manager reads the database, not the environment.
	reloaded := New(queries, envHash)
	if ok, _ := reloaded.Verify(ctx, "kopi-susu-2026"); !ok {
		t.Fatal("a password stored in the database must be visible to a new manager")
	}

	if err := manager.Set(ctx, "password-salah", "yang-lain-lagi"); err == nil {
		t.Fatal("changing the password with a wrong current one must be rejected")
	}
}
