// Package tools berisi logika bisnis ANABUL: validasi input, aturan bulan
// terkunci, dan perhitungan turunan. Dipakai bersama oleh REST API dan agent.
package tools

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"anabul/internal/store"
)

// Service membungkus store dengan aturan bisnis.
type Service struct {
	q    *store.Queries
	pool *pgxpool.Pool
	loc  *time.Location
}

func New(q *store.Queries, pool *pgxpool.Pool, loc *time.Location) *Service {
	if loc == nil {
		loc = time.FixedZone("WIB", 7*60*60)
	}
	return &Service{q: q, pool: pool, loc: loc}
}

// Location mengembalikan zona waktu aplikasi (Asia/Jakarta).
func (s *Service) Location() *time.Location { return s.loc }

// Now adalah waktu sekarang di zona aplikasi.
func (s *Service) Now() time.Time { return time.Now().In(s.loc) }

// CurrentMonth adalah bulan berjalan di zona aplikasi.
func (s *Service) CurrentMonth() time.Time { return MonthStart(s.Now(), s.loc) }

// ParseMonthArg membaca argumen bulan "YYYY-MM" (kosong = bulan berjalan).
func (s *Service) ParseMonthArg(raw string) (time.Time, error) {
	return ParseMonth(raw, s.Now(), s.loc)
}

// InTx menjalankan fn di dalam satu transaksi database.
func (s *Service) InTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("gagal rollback transaksi", "error", err)
		}
	}()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsLocked mengecek apakah bulan sudah ditutup (dan belum dibuka kembali).
func (s *Service) IsLocked(ctx context.Context, month time.Time) (bool, error) {
	closing, err := s.q.GetMonthClosing(ctx, dateOf(month))
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return IsLocked(closing.ClosedAt, closing.ReopenedAt), nil
}

// EnsureUnlocked menolak operasi tulis pada bulan yang sudah ditutup.
func (s *Service) EnsureUnlocked(ctx context.Context, month time.Time) error {
	isLocked, err := s.IsLocked(ctx, month)
	if err != nil {
		return err
	}
	if isLocked {
		return locked(FormatMonth(month))
	}
	return nil
}

// MonthLockInfo mengembalikan status kunci beberapa bulan sekaligus.
func (s *Service) MonthLockInfo(ctx context.Context, month time.Time) (bool, error) {
	return s.IsLocked(ctx, month)
}
