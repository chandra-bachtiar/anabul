// Package sched contains the cron jobs that generate monthly needs and close
// the month automatically.
package sched

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"anabul/internal/tools"
)

// Scheduler runs periodic jobs in the app timezone.
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

// Start registers the start-of-month job and runs one catch-up for the current
// month.
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
		slog.Error("failed to generate monthly needs", "month", tools.FormatMonth(month), "error", err)
		return
	}
	slog.Info("monthly needs ready", "month", tools.FormatMonth(month), "total", count)
}
