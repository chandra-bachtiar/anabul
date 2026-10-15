// Package tools holds ANABUL's business logic: input validation, the
// locked-month rule, and derived figures. Shared by the REST API and the agent.
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

func (s *Service) Location() *time.Location { return s.loc }

func (s *Service) Now() time.Time { return time.Now().In(s.loc) }

func (s *Service) CurrentMonth() time.Time { return MonthStart(s.Now(), s.loc) }

// ParseMonthArg reads a "YYYY-MM" argument; empty means the current month.
func (s *Service) ParseMonthArg(raw string) (time.Time, error) {
	return ParseMonth(raw, s.Now(), s.loc)
}

func (s *Service) InTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("transaction rollback failed", "error", err)
		}
	}()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsLocked reports whether the month is closed and has not been reopened since.
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

// EnsureUnlocked rejects writes into a month that is already closed.
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
