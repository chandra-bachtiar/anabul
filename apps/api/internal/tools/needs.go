package tools

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"anabul/internal/store"
)

// ListRecurringNeeds mengembalikan template kebutuhan bulanan.
func (s *Service) ListRecurringNeeds(ctx context.Context) ([]RecurringNeedRow, error) {
	rows, err := s.q.ListRecurringNeeds(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]RecurringNeedRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, RecurringNeedRow{
			ID:           row.ID,
			Name:         row.Name,
			Amount:       row.Amount,
			DueDay:       int(row.DueDay),
			CategoryID:   int8Value(row.CategoryID),
			CategoryName: textValueString(row.CategoryName),
			Active:       row.Active,
		})
	}
	return result, nil
}

// CreateRecurringNeed menambah template kebutuhan bulanan.
func (s *Service) CreateRecurringNeed(ctx context.Context, name string, amount int64, dueDay int, categoryID *int64) (RecurringNeedRow, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return RecurringNeedRow{}, invalid("nama kebutuhan belum diisi")
	}
	if amount < 0 {
		return RecurringNeedRow{}, invalid("nominal tidak boleh negatif")
	}
	if dueDay < 1 || dueDay > 31 {
		return RecurringNeedRow{}, invalid("tanggal jatuh tempo harus 1-31")
	}
	var categoryName string
	if categoryID != nil {
		category, err := s.getCategory(ctx, s.q, *categoryID)
		if err != nil {
			return RecurringNeedRow{}, err
		}
		categoryName = category.Name
	}
	row, err := s.q.CreateRecurringNeed(ctx, store.CreateRecurringNeedParams{
		Name:       name,
		Amount:     amount,
		DueDay:     int16(dueDay),
		CategoryID: int8Ptr(categoryID),
	})
	if err != nil {
		return RecurringNeedRow{}, err
	}
	return RecurringNeedRow{
		ID:           row.ID,
		Name:         row.Name,
		Amount:       row.Amount,
		DueDay:       int(row.DueDay),
		CategoryID:   int8Value(row.CategoryID),
		CategoryName: categoryName,
		Active:       row.Active,
	}, nil
}

// UpdateRecurringNeedInput adalah perubahan template kebutuhan.
type UpdateRecurringNeedInput struct {
	ID         int64
	Name       *string
	Amount     *int64
	DueDay     *int
	CategoryID *int64
	Active     *bool
}

// UpdateRecurringNeed mengubah template kebutuhan bulanan.
func (s *Service) UpdateRecurringNeed(ctx context.Context, in UpdateRecurringNeedInput) (RecurringNeedRow, error) {
	if in.Amount != nil && *in.Amount < 0 {
		return RecurringNeedRow{}, invalid("nominal tidak boleh negatif")
	}
	var dueDay *int16
	if in.DueDay != nil {
		if *in.DueDay < 1 || *in.DueDay > 31 {
			return RecurringNeedRow{}, invalid("tanggal jatuh tempo harus 1-31")
		}
		value := int16(*in.DueDay)
		dueDay = &value
	}
	row, err := s.q.UpdateRecurringNeed(ctx, store.UpdateRecurringNeedParams{
		ID:         in.ID,
		Name:       textPtr(in.Name),
		Amount:     int8Ptr(in.Amount),
		DueDay:     int16Ptr(dueDay),
		CategoryID: int8Ptr(in.CategoryID),
		Active:     boolPtr(in.Active),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RecurringNeedRow{}, notFound("kebutuhan tidak ditemukan")
		}
		return RecurringNeedRow{}, err
	}
	detail, err := s.q.GetRecurringNeed(ctx, row.ID)
	if err != nil {
		return RecurringNeedRow{}, err
	}
	return RecurringNeedRow{
		ID:           detail.ID,
		Name:         detail.Name,
		Amount:       detail.Amount,
		DueDay:       int(detail.DueDay),
		CategoryID:   int8Value(detail.CategoryID),
		CategoryName: textValueString(detail.CategoryName),
		Active:       detail.Active,
	}, nil
}

// DeleteRecurringNeed menghapus template tanpa menghapus riwayat bulan lalu.
func (s *Service) DeleteRecurringNeed(ctx context.Context, id int64) error {
	return s.InTx(ctx, func(q *store.Queries) error {
		if err := q.DetachRecurringNeedFromMonthlyNeeds(ctx, int8Ptr(&id)); err != nil {
			return err
		}
		return q.DeleteRecurringNeed(ctx, id)
	})
}

// GenerateMonthlyNeeds membuat daftar kebutuhan bulan dari template yang aktif.
func (s *Service) GenerateMonthlyNeeds(ctx context.Context, monthArg string) (int64, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return 0, err
	}
	if err := s.EnsureUnlocked(ctx, month); err != nil {
		return 0, err
	}
	if err := s.q.GenerateMonthlyNeeds(ctx, dateOf(month)); err != nil {
		return 0, err
	}
	return s.countNeeds(ctx, month)
}

func (s *Service) countNeeds(ctx context.Context, month time.Time) (int64, error) {
	rows, err := s.q.ListMonthlyNeeds(ctx, store.ListMonthlyNeedsParams{MonthStart: dateOf(month)})
	if err != nil {
		return 0, err
	}
	return int64(len(rows)), nil
}

// ListMonthlyNeeds mengembalikan kebutuhan bulanan pada satu bulan.
func (s *Service) ListMonthlyNeeds(ctx context.Context, monthArg, status string) ([]NeedRow, error) {
	month, err := s.ParseMonthArg(monthArg)
	if err != nil {
		return nil, err
	}
	return s.listMonthlyNeeds(ctx, month, status)
}

func (s *Service) listMonthlyNeeds(ctx context.Context, month time.Time, status string) ([]NeedRow, error) {
	rows, err := s.q.ListMonthlyNeeds(ctx, store.ListMonthlyNeedsParams{
		MonthStart: dateOf(month),
		Status:     textPtr(trimPtr(status)),
	})
	if err != nil {
		return nil, err
	}
	today := StartOfDay(s.Now(), s.loc)
	result := make([]NeedRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, needRow(row.ID, row.Name, row.Amount, row.DueDate, row.Status,
			row.TransactionID, row.CategoryID, row.CategoryName, today))
	}
	return result, nil
}

// MarkNeedPaid menandai kebutuhan lunas dan membuat transaksi pengeluarannya.
func (s *Service) MarkNeedPaid(ctx context.Context, needID int64, amount *int64) (NeedRow, error) {
	need, err := s.q.GetMonthlyNeed(ctx, needID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NeedRow{}, notFound("kebutuhan tidak ditemukan")
		}
		return NeedRow{}, err
	}
	if need.Status == "paid" {
		return NeedRow{}, &Error{Code: CodeConflict, Message: "kebutuhan " + need.Name + " sudah ditandai lunas"}
	}
	month := pgDateToTime(need.MonthStart, s.loc)
	if err := s.EnsureUnlocked(ctx, month); err != nil {
		return NeedRow{}, err
	}

	paidAmount := need.Amount
	if amount != nil {
		if *amount <= 0 {
			return NeedRow{}, invalid("nominal harus lebih dari 0")
		}
		paidAmount = *amount
	}

	var updated store.MonthlyNeed
	err = s.InTx(ctx, func(q *store.Queries) error {
		tx, txErr := s.createTransaction(ctx, q, AddTransactionInput{
			Type:       "expense",
			Amount:     paidAmount,
			Note:       need.Name,
			CategoryID: int8Value(need.CategoryID),
			Source:     "need",
		})
		if txErr != nil {
			return txErr
		}
		var markErr error
		updated, markErr = q.MarkMonthlyNeedPaid(ctx, store.MarkMonthlyNeedPaidParams{
			ID:            need.ID,
			TransactionID: pgtype.Int8{Int64: tx.ID, Valid: true},
		})
		return markErr
	})
	if err != nil {
		return NeedRow{}, err
	}

	dueDate := pgtype.Date{Time: updated.DueDate.Time, Valid: updated.DueDate.Valid}
	return needRow(updated.ID, updated.Name, updated.Amount, dueDate, updated.Status,
		updated.TransactionID, need.CategoryID, need.CategoryName, StartOfDay(s.Now(), s.loc)), nil
}

// UnmarkNeedPaid membatalkan status lunas (transaksi yang dibuat tidak dihapus).
func (s *Service) UnmarkNeedPaid(ctx context.Context, needID int64) (NeedRow, error) {
	need, err := s.q.GetMonthlyNeed(ctx, needID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NeedRow{}, notFound("kebutuhan tidak ditemukan")
		}
		return NeedRow{}, err
	}
	if err := s.EnsureUnlocked(ctx, pgDateToTime(need.MonthStart, s.loc)); err != nil {
		return NeedRow{}, err
	}
	if _, err := s.q.MarkMonthlyNeedUnpaid(ctx, needID); err != nil {
		return NeedRow{}, err
	}
	updated, err := s.q.GetMonthlyNeed(ctx, needID)
	if err != nil {
		return NeedRow{}, err
	}
	return needRow(updated.ID, updated.Name, updated.Amount, updated.DueDate, updated.Status,
		updated.TransactionID, updated.CategoryID, updated.CategoryName, StartOfDay(s.Now(), s.loc)), nil
}

// FindNeedByName mencari kebutuhan pada bulan berjalan berdasarkan nama.
func (s *Service) FindNeedByName(ctx context.Context, month time.Time, name string) (NeedRow, error) {
	row, err := s.q.FindMonthlyNeedByName(ctx, store.FindMonthlyNeedByNameParams{
		MonthStart: dateOf(month),
		Lower:      strings.TrimSpace(name),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NeedRow{}, notFound("kebutuhan %q tidak ada di bulan ini", name)
		}
		return NeedRow{}, err
	}
	return needRow(row.ID, row.Name, row.Amount, row.DueDate, row.Status,
		row.TransactionID, row.CategoryID, row.CategoryName, StartOfDay(s.Now(), s.loc)), nil
}

func (s *Service) needsTotals(ctx context.Context, month time.Time) (int64, int64, error) {
	rows, err := s.q.ListMonthlyNeeds(ctx, store.ListMonthlyNeedsParams{
		MonthStart: dateOf(month),
		Status:     pgtype.Text{String: "unpaid", Valid: true},
	})
	if err != nil {
		return 0, 0, err
	}
	var total int64
	for _, row := range rows {
		total += row.Amount
	}
	return int64(len(rows)), total, nil
}

func needRow(id int64, name string, amount int64, dueDate pgtype.Date, status string,
	transactionID, categoryID pgtype.Int8, categoryName pgtype.Text, today time.Time) NeedRow {
	row := NeedRow{
		ID:            id,
		Name:          name,
		Amount:        amount,
		DueDate:       "",
		Status:        status,
		CategoryID:    int8Value(categoryID),
		CategoryName:  textValueString(categoryName),
		TransactionID: int8Value(transactionID),
	}
	if dueDate.Valid {
		due := time.Date(dueDate.Time.Year(), dueDate.Time.Month(), dueDate.Time.Day(), 0, 0, 0, 0, today.Location())
		row.DueDate = due.Format("2006-01-02")
		if status == "unpaid" {
			row.Overdue = due.Before(today)
			row.DueSoon = !row.Overdue && !due.After(today.AddDate(0, 0, 3))
		}
	}
	return row
}
