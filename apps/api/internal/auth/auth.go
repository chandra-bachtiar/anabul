// Package auth stores and checks the app password. The first hash comes from
// the environment and is copied into app_settings, so the password can later be
// changed from inside the app.
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

// DefaultPassword is the sample password used during development and quoted in
// the README. While it is still active the app keeps suggesting a change.
const DefaultPassword = "anabul123"

const MinPasswordLength = 8

const settingKey = "password_hash"

type Manager struct {
	q       *store.Queries
	envHash string

	mu     sync.RWMutex
	cached string
}

func New(q *store.Queries, envHash string) *Manager {
	return &Manager{q: q, envHash: strings.TrimSpace(envHash)}
}

// EnsureSeeded copies the environment hash into the database when it is missing.
// It reports whether a new row was written.
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
	m.store(m.envHash)
	return true, nil
}

// hash returns the active hash: the database first, then the environment.
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
		m.store(value)
		return value, nil
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return "", err
	}

	// Not in the database yet: fall back to the environment value.
	if m.envHash == "" {
		return "", nil
	}
	if err := m.q.UpsertSetting(ctx, store.UpsertSettingParams{Key: settingKey, Value: m.envHash}); err != nil {
		return "", err
	}
	m.store(m.envHash)
	return m.envHash, nil
}

func (m *Manager) store(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cached = hash
}

func (m *Manager) Configured(ctx context.Context) (bool, error) {
	hash, err := m.hash(ctx)
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

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

// UsingDefault reports whether the active password is still DefaultPassword.
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

// Set changes the password after checking the current one.
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
	m.store(string(hash))
	return nil
}

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
