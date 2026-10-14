// Package auth menyimpan dan memeriksa password aplikasi. Password awal
// diambil dari environment, lalu salinannya disimpan di tabel app_settings
// supaya bisa diganti dari dalam aplikasi.
package auth

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"anabul/internal/store"
	"anabul/internal/tools"
)

// DefaultPassword adalah password contoh yang dipakai saat pengembangan dan
// ikut ditulis di README. Selama masih dipakai, aplikasi menyarankan ganti.
const DefaultPassword = "anabul123"

// MinPasswordLength adalah panjang minimal password baru.
const MinPasswordLength = 8

const settingKey = "password_hash"

type Manager struct {
	q       *store.Queries
	envHash string

	mu     sync.RWMutex
	cached string
	aktif  bool
}

func New(q *store.Queries, envHash string) *Manager {
	return &Manager{q: q, envHash: strings.TrimSpace(envHash)}
}

// EnsureSeeded menyalin hash dari environment ke database kalau belum ada.
// Mengembalikan true kalau baris baru dibuat.
func (m *Manager) EnsureSeeded(ctx context.Context) (bool, error) {
	if m.envHash == "" {
		return false, nil
	}
	if _, err := m.q.GetSetting(ctx, settingKey); err == nil {
		return false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	if err := m.q.UpsertSetting(ctx, store.UpsertSettingParams{Key: settingKey, Value: m.envHash}); err != nil {
		return false, err
	}
	m.simpan(m.envHash)
	return true, nil
}

// hash mengembalikan hash aktif: database dulu, lalu environment.
func (m *Manager) hash(ctx context.Context) (string, error) {
	m.mu.RLock()
	cached := m.cached
	m.mu.RUnlock()
	if cached != "" {
		return cached, nil
	}

	value, err := m.q.GetSetting(ctx, settingKey)
	switch {
	case err == nil && strings.TrimSpace(value) != "":
		m.simpan(value)
		return value, nil
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return "", err
	}

	// Belum ada di database: pakai nilai dari environment.
	if m.envHash == "" {
		return "", nil
	}
	if err := m.q.UpsertSetting(ctx, store.UpsertSettingParams{Key: settingKey, Value: m.envHash}); err != nil {
		return "", err
	}
	m.simpan(m.envHash)
	return m.envHash, nil
}

func (m *Manager) simpan(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cached = hash
	m.aktif = bcrypt.CompareHashAndPassword([]byte(hash), []byte(DefaultPassword)) == nil
}

// Configured menandai apakah password sudah diatur.
func (m *Manager) Configured(ctx context.Context) (bool, error) {
	hash, err := m.hash(ctx)
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

// Verify mencocokkan password dengan hash aktif.
func (m *Manager) Verify(ctx context.Context, password string) (bool, error) {
	hash, err := m.hash(ctx)
	if err != nil {
		return false, err
	}
	if hash == "" {
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, nil
}

// UsingDefault menandai password yang masih sama dengan DefaultPassword.
func (m *Manager) UsingDefault(ctx context.Context) (bool, error) {
	hash, err := m.hash(ctx)
	if err != nil {
		return false, err
	}
	if hash == "" {
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(DefaultPassword)) == nil, nil
}

// Set mengganti password setelah password lama diperiksa.
func (m *Manager) Set(ctx context.Context, current, next string) error {
	ok, err := m.Verify(ctx, current)
	if err != nil {
		return err
	}
	if !ok {
		return tools.NewInvalidError("password sekarang salah")
	}
	if err := ValidateNewPassword(current, next); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := m.q.UpsertSetting(ctx, store.UpsertSettingParams{Key: settingKey, Value: string(hash)}); err != nil {
		return err
	}
	m.simpan(string(hash))
	return nil
}

// ValidateNewPassword memeriksa aturan password baru.
func ValidateNewPassword(current, next string) error {
	next = strings.TrimSpace(next)
	switch {
	case next == "":
		return tools.NewInvalidError("password baru belum diisi")
	case len(next) < MinPasswordLength:
		return tools.NewInvalidError("password baru minimal %d karakter", MinPasswordLength)
	case next == current:
		return tools.NewInvalidError("password baru masih sama dengan yang sekarang")
	case next == DefaultPassword:
		return tools.NewInvalidError("jangan pakai password default %q", DefaultPassword)
	default:
		return nil
	}
}
