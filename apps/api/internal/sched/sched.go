// Package sched berisi penjadwalan (cron) untuk men-generate kebutuhan
// bulanan dan menutup bulan secara otomatis.
package sched

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"anabul/internal/tools"
)

// Scheduler menjalankan tugas berkala di zona waktu aplikasi.
type Scheduler struct {
	tools *tools.Service
	cron  *cron.Cron
}

func New(svc *tools.Service) *Scheduler {
	location := svc.Location()
	return &Scheduler{
		tools: svc,
		cron:  cron.New(cron.WithLocation(location)),
	}
}

// Start menjalankan tugas awal bulan dan sekali catch-up untuk bulan berjalan.
func (s *Scheduler) Start(ctx context.Context) error {
	if _, err := s.cron.AddFunc("5 0 1 * *", func() {
		s.generateMonth(context.WithoutCancel(ctx), s.tools.CurrentMonth())
	}); err != nil {
		return err
	}
	s.cron.Start()

	s.generateMonth(ctx, s.tools.CurrentMonth())
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		<-s.cron.Stop().Done()
	}
}

func (s *Scheduler) generateMonth(ctx context.Context, month time.Time) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	count, err := s.tools.GenerateMonthlyNeeds(ctx, tools.FormatMonth(month))
	if err != nil {
		slog.Error("gagal menyiapkan kebutuhan bulanan", "month", tools.FormatMonth(month), "error", err)
		return
	}
	slog.Info("kebutuhan bulanan siap", "month", tools.FormatMonth(month), "total", count)
}
