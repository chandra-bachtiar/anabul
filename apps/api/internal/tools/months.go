package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"anabul/internal/store"
)

func (s *Service) MonthSummary(ctx context.Context, monthArg string) (Summary, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return Summary{}, err
	}
	return s.buildSummary(ctx, month)
}

func (s *Service) MonthPreview(ctx context.Context, monthArg string) (MonthPreview, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return MonthPreview{}, err
	}

	summary, err := s.buildSummary(ctx, month)
	if err != nil {
		return MonthPreview{}, err
	}

	unpaid, err := s.listMonthlyNeeds(ctx, month, "unpaid")
	if err != nil {
		return MonthPreview{}, err
	}

	openDebts, err := s.ListDebts(ctx, "open", "")
	if err != nil {
		return MonthPreview{}, err
	}
	dueDebts := make([]DebtRow, 0)
	for _, debt := range openDebts {
		if debt.DueDate != nil {
			dueDebts = append(dueDebts, debt)
		}
	}

	overBudget := make([]BudgetRow, 0)
	for _, row := range summary.Budgets {
		if row.Remaining < 0 {
			overBudget = append(overBudget, row)
		}
	}

	warnings := make([]string, 0, 3)
	if len(overBudget) > 0 {
		warnings = append(warnings, "ada "+strconv.Itoa(len(overBudget))+" kategori yang lewat budget")
	}
	if len(unpaid) > 0 {
		warnings = append(warnings, "masih ada "+strconv.Itoa(len(unpaid))+" kebutuhan belum dibayar")
	}
	if summary.DebtTotals.Hutang > 0 {
		warnings = append(warnings, "hutang berjalan "+FormatAmount(summary.DebtTotals.Hutang))
	}

	preview := MonthPreview{
		Summary:        summary,
		UnpaidNeedRows: unpaid,
		OpenDebts:      openDebts,
		DueDebts:       dueDebts,
		BudgetOver:     overBudget,
		Warnings:       warnings,
	}
	preview.Snapshot = s.snapshotOf(summary, unpaid)
	return preview, nil
}

// CloseMonth locks the month, stores its snapshot, and carries budgets and
// recurring needs into the next month in the same transaction.
func (s *Service) CloseMonth(ctx context.Context, monthArg string) (Snapshot, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return Snapshot{}, err
	}

	isLocked, err := s.IsLocked(ctx, month)
	if err != nil {
		return Snapshot{}, err
	}
	if isLocked {
		return Snapshot{}, &Error{Code: CodeConflict, Message: "bulan " + FormatMonth(month) + " sudah ditutup"}
	}

	summary, err := s.buildSummary(ctx, month)
	if err != nil {
		return Snapshot{}, err
	}
	unpaid, err := s.listMonthlyNeeds(ctx, month, "unpaid")
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := s.snapshotOf(summary, unpaid)
	snapshot.ClosedAt = s.Now()

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, err
	}

	next := MonthEnd(month)
	err = s.InTx(ctx, func(q *store.Queries) error {
		if _, err := q.UpsertMonthClosing(ctx, store.UpsertMonthClosingParams{
			MonthStart: dateOf(month),
			Snapshot:   payload,
		}); err != nil {
			return err
		}
		if err := q.CopyBudgets(ctx, store.CopyBudgetsParams{
			MonthStart: dateOf(month),
			Column2:    dateOf(next),
		}); err != nil {
			return err
		}
		return q.GenerateMonthlyNeeds(ctx, dateOf(next))
	})
	if err != nil {
		return Snapshot{}, err
	}

	nextBudgets, err := s.listBudgets(ctx, next)
	if err != nil {
		return Snapshot{}, err
	}
	nextNeeds, err := s.countNeeds(ctx, next)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.CarriedBudgets = len(nextBudgets)
	snapshot.CarriedNeeds = int(nextNeeds)
	return snapshot, nil
}

func (s *Service) ReopenMonth(ctx context.Context, monthArg string) error {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return err
	}
	isLocked, err := s.IsLocked(ctx, month)
	if err != nil {
		return err
	}
	if !isLocked {
		return &Error{Code: CodeConflict, Message: "bulan " + FormatMonth(month) + " memang belum ditutup"}
	}
	if _, err := s.q.ReopenMonthClosing(ctx, dateOf(month)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound("catatan tutup bulan tidak ditemukan")
		}
		return err
	}
	return nil
}

func (s *Service) ClosedMonths(ctx context.Context) ([]string, error) {
	rows, err := s.q.ListMonthClosings(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, FormatMonth(pgDateToTime(row.MonthStart, s.loc)))
	}
	return result, nil
}

func (s *Service) snapshotOf(summary Summary, unpaid []NeedRow) Snapshot {
	return Snapshot{
		Month:            summary.Month,
		MonthLabel:       summary.MonthLabel,
		ClosedAt:         s.Now(),
		Income:           summary.Income,
		Expense:          summary.Expense,
		Net:              summary.Net,
		BudgetTotal:      summary.BudgetTotal,
		BudgetSpent:      summary.BudgetSpent,
		Budgets:          summary.Budgets,
		Categories:       summary.Categories,
		UnpaidNeeds:      unpaid,
		DebtTotals:       summary.DebtTotals,
		TransactionCount: summary.transactionCount,
	}
}
