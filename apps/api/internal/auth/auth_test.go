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
		{name: "kosong", current: "anabul123", next: "   ", wantErr: true, contains: "belum diisi"},
		{name: "terlalu pendek", current: "anabul123", next: "pendek", wantErr: true, contains: "minimal"},
		{name: "sama dengan sekarang", current: "rahasia-lama", next: "rahasia-lama", wantErr: true, contains: "masih sama"},
		{name: "masih password default", current: "rahasia-lama", next: DefaultPassword, wantErr: true, contains: "default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNewPassword(tc.current, tc.next)
			if tc.wantErr && err == nil {
				t.Fatal("ingin error, tapi lolos")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ingin lolos, tapi error: %v", err)
			}
			if tc.contains != "" && (err == nil || !strings.Contains(err.Error(), tc.contains)) {
				t.Fatalf("pesan error %v tidak memuat %q", err, tc.contains)
			}
		})
	}
}

// TestManagerGantiPassword memakai database uji; tanpa ANABUL_TEST_DATABASE_URL
// test dilewati supaya `make test` tetap jalan tanpa DB.
func TestManagerGantiPassword(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ANABUL_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ANABUL_TEST_DATABASE_URL belum diisi, test ganti password dilewati")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("koneksi database uji: %v", err)
	}
	// t.Cleanup berjalan terbalik (LIFO): tutup pool didaftarkan lebih dulu
	// supaya pembersihan data berjalan sebelum koneksi ditutup.
	t.Cleanup(pool.Close)

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrasi database uji: %v", err)
	}

	// Mulai dari kondisi bersih, dan kembalikan lagi setelah test supaya
	// aplikasi kembali memakai hash dari environment.
	if _, err := pool.Exec(ctx, "DELETE FROM app_settings WHERE key = $1", settingKey); err != nil {
		t.Fatalf("bersihkan app_settings: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM app_settings WHERE key = $1", settingKey); err != nil {
			t.Errorf("gagal membersihkan app_settings: %v", err)
		}
	})

	queries := store.New(pool)
	envHash := "$2a$10$tJTejoXklz5EivrCcRa/DuJc/ojCxtGB.PWzaWeGLSapcZkNbS3ga" // hash dari "anabul123"

	manager := New(queries, envHash)
	seeded, err := manager.EnsureSeeded(ctx)
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}
	if !seeded {
		t.Fatal("hash dari environment harus disalin ke database saat pertama")
	}

	configured, err := manager.Configured(ctx)
	if err != nil || !configured {
		t.Fatalf("Configured = %v (err %v), ingin true", configured, err)
	}

	usingDefault, err := manager.UsingDefault(ctx)
	if err != nil || !usingDefault {
		t.Fatalf("UsingDefault = %v (err %v), ingin true sebelum diganti", usingDefault, err)
	}

	if ok, _ := manager.Verify(ctx, "salah"); ok {
		t.Fatal("password salah tidak boleh lolos")
	}
	if ok, _ := manager.Verify(ctx, DefaultPassword); !ok {
		t.Fatal("password default harus lolos sebelum diganti")
	}

	if err := manager.Set(ctx, DefaultPassword, "kopi-susu-2026"); err != nil {
		t.Fatalf("ganti password: %v", err)
	}

	if ok, _ := manager.Verify(ctx, DefaultPassword); ok {
		t.Fatal("password lama tidak boleh dipakai lagi")
	}
	if ok, _ := manager.Verify(ctx, "kopi-susu-2026"); !ok {
		t.Fatal("password baru harus lolos")
	}
	if usingDefault, _ := manager.UsingDefault(ctx); usingDefault {
		t.Fatal("setelah diganti, UsingDefault harus false")
	}

	// Manager baru membaca dari database, bukan dari environment.
	ulang := New(queries, envHash)
	if ok, _ := ulang.Verify(ctx, "kopi-susu-2026"); !ok {
		t.Fatal("password tersimpan di database harus terbaca manager baru")
	}

	if err := manager.Set(ctx, "password-salah", "yang-lain-lagi"); err == nil {
		t.Fatal("ganti password dengan password lama yang salah harus ditolak")
	}
}
