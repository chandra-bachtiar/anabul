package tools

import (
	"context"
	"sort"
	"time"
)

// Dashboard merangkai semua yang dibutuhkan halaman Home.
func (s *Service) Dashboard(ctx context.Context, monthArg string) (Dashboard, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return Dashboard{}, err
	}

	summary, err := s.buildSummary(ctx, month)
	if err != nil {
		return Dashboard{}, err
	}

	recent, err := s.LastTransactions(ctx, month, 5)
	if err != nil {
		return Dashboard{}, err
	}
	needs, err := s.listMonthlyNeeds(ctx, month, "unpaid")
	if err != nil {
		return Dashboard{}, err
	}

	top := make([]CategorySpend, 0, 5)
	for _, row := range summary.Categories {
		if row.Spent == 0 {
			continue
		}
		top = append(top, row)
		if len(top) == 5 {
			break
		}
	}

	return Dashboard{
		Month:            FormatMonth(month),
		MonthLabel:       FormatMonthLabel(month),
		Locked:           summary.Locked,
		Mood:             Mood(summary.Locked, summary.BudgetTotal, summary.BudgetSpent, anyOver(summary.Budgets)),
		Income:           summary.Income,
		Expense:          summary.Expense,
		Net:              summary.Net,
		BudgetTotal:      summary.BudgetTotal,
		BudgetSpent:      summary.BudgetSpent,
		BudgetRemaining:  summary.BudgetLeft,
		BudgetStatus:     BudgetStatus(summary.BudgetTotal, summary.BudgetSpent),
		Budgets:          summary.Budgets,
		TopCategories:    top,
		Needs:            needs,
		NeedsUnpaid:      summary.UnpaidNeeds,
		NeedsUnpaidTotal: summary.NeedsTotal,
		Debts:            summary.DebtTotals,
		Recent:           recent,
		TransactionCount: summary.transactionCount,
	}, nil
}

func (s *Service) buildSummary(ctx context.Context, month time.Time) (Summary, error) {
	totals, err := s.q.MonthTotals(ctx, dateOf(month))
	if err != nil {
		return Summary{}, err
	}
	overview, err := s.budgetOverview(ctx, month)
	if err != nil {
		return Summary{}, err
	}
	categories, err := s.categorySpendRows(ctx, month)
	if err != nil {
		return Summary{}, err
	}
	unpaidCount, unpaidTotal, err := s.needsTotals(ctx, month)
	if err != nil {
		return Summary{}, err
	}
	debts, err := s.DebtTotals(ctx)
	if err != nil {
		return Summary{}, err
	}
	isLocked, err := s.IsLocked(ctx, month)
	if err != nil {
		return Summary{}, err
	}

	sum := Summary{
		Month:       FormatMonth(month),
		MonthLabel:  FormatMonthLabel(month),
		Locked:      isLocked,
		Income:      totals.Income,
		Expense:     totals.Expense,
		Net:         totals.Income - totals.Expense,
		BudgetTotal: overview.Total,
		BudgetSpent: overview.Spent,
		BudgetLeft:  overview.Remaining,
		Categories:  sortBySpent(categories),
		UnpaidNeeds: unpaidCount,
		NeedsTotal:  unpaidTotal,
		DebtTotals:  debts,
		Profitable:  totals.Income >= totals.Expense,
	}
	sum.transactionCount = totals.TxCount
	sum.Budgets = overview.Rows
	sum.BiggestDrain = topN(sum.Categories, 3)
	if len(sum.Categories) > 0 && sum.Categories[0].Spent > 0 {
		sum.TopCategory = sum.Categories[0].CategoryName
		sum.TopSpent = sum.Categories[0].Spent
	}
	return sum, nil
}

func sortBySpent(rows []CategorySpend) []CategorySpend {
	sorted := make([]CategorySpend, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Spent == sorted[j].Spent {
			return sorted[i].CategoryName < sorted[j].CategoryName
		}
		return sorted[i].Spent > sorted[j].Spent
	})
	return sorted
}

func topN(rows []CategorySpend, n int) []CategorySpend {
	result := make([]CategorySpend, 0, n)
	for _, row := range rows {
		if row.Spent == 0 {
			continue
		}
		result = append(result, row)
		if len(result) == n {
			break
		}
	}
	return result
}

func anyOver(rows []BudgetRow) bool {
	for _, row := range rows {
		if row.Status == StatusOver || row.Remaining < 0 {
			return true
		}
	}
	return false
}
