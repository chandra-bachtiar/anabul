package tools

import (
	"context"
	"time"

	"anabul/internal/store"
)

// ListBudgets mengembalikan budget satu bulan beserta pemakaiannya.
func (s *Service) ListBudgets(ctx context.Context, monthArg string) ([]BudgetRow, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return nil, err
	}
	return s.listBudgets(ctx, month)
}

func (s *Service) listBudgets(ctx context.Context, month time.Time) ([]BudgetRow, error) {
	rows, err := s.q.ListBudgets(ctx, dateOf(month))
	if err != nil {
		return nil, err
	}
	result := make([]BudgetRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, budgetRow(row.CategoryID, row.CategoryName, row.Amount, row.Spent))
	}
	return result, nil
}

// SetBudget menyimpan nominal budget satu kategori pada satu bulan.
func (s *Service) SetBudget(ctx context.Context, monthArg string, categoryID int64, amount int64) (BudgetRow, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return BudgetRow{}, err
	}
	if amount < 0 {
		return BudgetRow{}, invalid("nominal budget tidak boleh negatif")
	}
	if err := s.EnsureUnlocked(ctx, month); err != nil {
		return BudgetRow{}, err
	}
	category, err := s.getCategory(ctx, s.q, categoryID)
	if err != nil {
		return BudgetRow{}, err
	}
	return s.upsertBudget(ctx, month, category.ID, category.Name, amount)
}

func (s *Service) upsertBudget(ctx context.Context, month time.Time, categoryID int64, categoryName string, amount int64) (BudgetRow, error) {
	if _, err := s.q.UpsertBudget(ctx, store.UpsertBudgetParams{
		MonthStart: dateOf(month),
		CategoryID: categoryID,
		Amount:     amount,
	}); err != nil {
		return BudgetRow{}, err
	}
	spent, err := s.q.CategorySpent(ctx, store.CategorySpentParams{
		MonthStart: dateOf(month),
		CategoryID: int8Ptr(&categoryID),
	})
	if err != nil {
		return BudgetRow{}, err
	}
	return budgetRow(categoryID, categoryName, amount, spent), nil
}

// CopyBudgets menyalin budget dari satu bulan ke bulan lain.
func (s *Service) CopyBudgets(ctx context.Context, fromArg, toArg string) (int64, error) {
	from, err := s.ParseMonthArg(fromArg)
	if err != nil {
		return 0, err
	}
	to, err := s.ParseMonthArg(toArg)
	if err != nil {
		return 0, err
	}
	if err := s.EnsureUnlocked(ctx, to); err != nil {
		return 0, err
	}
	if err := s.q.CopyBudgets(ctx, store.CopyBudgetsParams{
		MonthStart: dateOf(from),
		Column2:    dateOf(to),
	}); err != nil {
		return 0, err
	}
	rows, err := s.listBudgets(ctx, to)
	if err != nil {
		return 0, err
	}
	return int64(len(rows)), nil
}

// BudgetOverview adalah total budget dan pemakaiannya pada satu bulan.
type BudgetOverview struct {
	Total        int64
	Spent        int64
	Remaining    int64
	Status       string
	Rows         []BudgetRow
	AnyOver      bool
	Unbudgeted   int64
	UnbudgetedBy []CategorySpend
}

func (s *Service) budgetOverview(ctx context.Context, month time.Time) (BudgetOverview, error) {
	rows, err := s.listBudgets(ctx, month)
	if err != nil {
		return BudgetOverview{}, err
	}

	spentByCategory, err := s.q.MonthExpenseByCategory(ctx, dateOf(month))
	if err != nil {
		return BudgetOverview{}, err
	}

	budgetByCategory := make(map[int64]BudgetRow, len(rows))
	overview := BudgetOverview{Rows: rows}
	for _, row := range rows {
		budgetByCategory[row.CategoryID] = row
		overview.Total += row.Amount
		if row.Amount > 0 {
			overview.Spent += row.Spent
		}
		if row.Remaining < 0 || row.Status == StatusOver {
			overview.AnyOver = true
		}
	}
	overview.Remaining = overview.Total - overview.Spent
	overview.Status = BudgetStatus(overview.Total, overview.Spent)

	for _, spend := range spentByCategory {
		var categoryID int64
		if spend.CategoryID.Valid {
			categoryID = spend.CategoryID.Int64
		}
		if _, ok := budgetByCategory[categoryID]; ok {
			continue
		}
		overview.Unbudgeted += spend.Spent
		overview.UnbudgetedBy = append(overview.UnbudgetedBy, CategorySpend{
			CategoryID:   int8Value(spend.CategoryID),
			CategoryName: categoryNameOrLabel(spend.CategoryName),
			Spent:        spend.Spent,
			Status:       StatusNone,
		})
	}
	return overview, nil
}

// categorySpendRows menggabungkan pengeluaran bulan dengan budget yang ada.
func (s *Service) categorySpendRows(ctx context.Context, month time.Time) ([]CategorySpend, error) {
	overview, err := s.budgetOverview(ctx, month)
	if err != nil {
		return nil, err
	}

	result := make([]CategorySpend, 0, len(overview.Rows)+len(overview.UnbudgetedBy))
	for _, row := range overview.Rows {
		result = append(result, CategorySpend{
			CategoryID:   &row.CategoryID,
			CategoryName: row.CategoryName,
			Spent:        row.Spent,
			Budget:       row.Amount,
			Remaining:    row.Remaining,
			Status:       row.Status,
			Progress:     row.Progress,
		})
	}
	result = append(result, overview.UnbudgetedBy...)
	return result, nil
}

func budgetRow(categoryID int64, categoryName string, amount, spent int64) BudgetRow {
	remaining := BudgetRemaining(amount, spent)
	return BudgetRow{
		CategoryID:   categoryID,
		CategoryName: categoryName,
		Amount:       amount,
		Spent:        spent,
		Remaining:    remaining,
		Status:       BudgetStatus(amount, spent),
		Progress:     progress(amount, spent),
	}
}

func progress(amount, spent int64) int {
	if amount <= 0 {
		return 0
	}
	value := int(spent * 100 / amount)
	if value < 0 {
		return 0
	}
	return value
}

func categoryNameOrLabel(name string) string {
	if name == "" {
		return "tanpa kategori"
	}
	return name
}
