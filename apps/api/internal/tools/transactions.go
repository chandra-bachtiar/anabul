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

// AddTransactionInput adalah parameter lengkap untuk mencatat transaksi.
type AddTransactionInput struct {
	Type       string
	Amount     int64
	Note       string
	CategoryID *int64
	Category   string
	AccountID  *int64
	Account    string
	OccurredAt *time.Time
	Source     string
}

// UpdateTransactionInput memakai pointer supaya field yang tidak dikirim tidak
// ikut berubah.
type UpdateTransactionInput struct {
	ID            int64
	Type          *string
	Amount        *int64
	Note          *string
	CategoryID    *int64
	ClearCategory bool
	AccountID     *int64
	ClearAccount  bool
	OccurredAt    *time.Time
}

// TransactionFilter adalah filter daftar transaksi.
type TransactionFilter struct {
	Month      string
	CategoryID *int64
	Query      string
	Cursor     *int64
	Limit      int32
}

// AddTransaction mencatat pemasukan/pengeluaran baru.
func (s *Service) AddTransaction(ctx context.Context, in AddTransactionInput) (Transaction, error) {
	return s.createTransaction(ctx, s.q, in)
}

func (s *Service) createTransaction(ctx context.Context, q *store.Queries, in AddTransactionInput) (Transaction, error) {
	txType := strings.ToLower(strings.TrimSpace(in.Type))
	if txType == "" {
		txType = "expense"
	}
	if txType != "expense" && txType != "income" {
		return Transaction{}, invalid("tipe harus 'expense' atau 'income'")
	}
	if in.Amount <= 0 {
		return Transaction{}, invalid("nominal harus lebih dari 0")
	}

	occurredAt := s.Now()
	if in.OccurredAt != nil {
		occurredAt = in.OccurredAt.In(s.loc)
	}
	month := MonthStart(occurredAt, s.loc)
	if err := s.EnsureUnlocked(ctx, month); err != nil {
		return Transaction{}, err
	}

	category, err := s.resolveCategory(ctx, q, in.CategoryID, in.Category, in.Note)
	if err != nil {
		return Transaction{}, err
	}
	account, err := s.resolveAccount(ctx, q, in.AccountID, in.Account)
	if err != nil {
		return Transaction{}, err
	}

	source := in.Source
	if source == "" {
		source = "manual"
	}

	row, err := q.CreateTransaction(ctx, store.CreateTransactionParams{
		Type:       txType,
		Amount:     in.Amount,
		Note:       strings.TrimSpace(in.Note),
		CategoryID: int8Ptr(categoryIDPtr(category)),
		AccountID:  int8Ptr(accountIDPtr(account)),
		OccurredAt: timeOfPtr(&occurredAt),
		MonthStart: dateOf(month),
		Source:     source,
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.transactionFromStore(row, category, account), nil
}

// UpdateTransaction mengubah transaksi yang sudah ada.
func (s *Service) UpdateTransaction(ctx context.Context, in UpdateTransactionInput) (Transaction, error) {
	current, err := s.q.GetTransaction(ctx, in.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, notFound("transaksi tidak ditemukan")
		}
		return Transaction{}, err
	}

	if err := s.EnsureUnlocked(ctx, pgDateToTime(current.MonthStart, s.loc)); err != nil {
		return Transaction{}, err
	}
	if in.Amount != nil && *in.Amount <= 0 {
		return Transaction{}, invalid("nominal harus lebih dari 0")
	}
	if in.Type != nil {
		value := strings.ToLower(strings.TrimSpace(*in.Type))
		if value != "expense" && value != "income" {
			return Transaction{}, invalid("tipe harus 'expense' atau 'income'")
		}
		in.Type = &value
	}
	if in.CategoryID != nil {
		if _, err := s.getCategory(ctx, s.q, *in.CategoryID); err != nil {
			return Transaction{}, err
		}
	}
	if in.AccountID != nil {
		if _, err := s.getAccount(ctx, s.q, *in.AccountID); err != nil {
			return Transaction{}, err
		}
	}

	var monthStart *time.Time
	if in.OccurredAt != nil {
		occurredAt := in.OccurredAt.In(s.loc)
		in.OccurredAt = &occurredAt
		month := MonthStart(occurredAt, s.loc)
		if err := s.EnsureUnlocked(ctx, month); err != nil {
			return Transaction{}, err
		}
		monthStart = &month
	}

	row, err := s.q.UpdateTransaction(ctx, store.UpdateTransactionParams{
		ID:            in.ID,
		Type:          textPtr(in.Type),
		Amount:        int8Ptr(in.Amount),
		Note:          textPtr(in.Note),
		CategoryID:    int8Ptr(in.CategoryID),
		ClearCategory: in.ClearCategory,
		AccountID:     int8Ptr(in.AccountID),
		ClearAccount:  in.ClearAccount,
		OccurredAt:    timeOfPtr(in.OccurredAt),
		MonthStart:    dateOfPtr(monthStart),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, notFound("transaksi tidak ditemukan")
		}
		return Transaction{}, err
	}

	updated, err := s.q.GetTransaction(ctx, row.ID)
	if err != nil {
		return Transaction{}, err
	}
	return s.transactionFromGetRow(updated), nil
}

// DeleteTransaction menghapus transaksi (soft delete).
func (s *Service) DeleteTransaction(ctx context.Context, id int64) error {
	current, err := s.q.GetTransaction(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound("transaksi tidak ditemukan")
		}
		return err
	}
	if err := s.EnsureUnlocked(ctx, pgDateToTime(current.MonthStart, s.loc)); err != nil {
		return err
	}
	return s.q.SoftDeleteTransaction(ctx, id)
}

// LastTransaction mengambil transaksi terakhir (untuk "yang tadi salah").
func (s *Service) LastTransaction(ctx context.Context) (Transaction, error) {
	row, err := s.q.LastTransaction(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, notFound("belum ada transaksi")
		}
		return Transaction{}, err
	}
	return s.transactionFromLastRow(row), nil
}

// ListTransactions mengembalikan transaksi satu bulan dengan filter opsional.
func (s *Service) ListTransactions(ctx context.Context, filter TransactionFilter) ([]Transaction, error) {
	month, err := s.ParseMonthArg(filter.Month)
	if err != nil {
		return nil, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.q.ListTransactions(ctx, store.ListTransactionsParams{
		MonthStart: dateOf(month),
		CategoryID: int8Ptr(filter.CategoryID),
		Q:          textPtr(trimPtr(filter.Query)),
		Cursor:     int8Ptr(filter.Cursor),
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		result = append(result, s.transactionFromListRow(row))
	}
	return result, nil
}

// LastTransactions mengambil beberapa transaksi terakhir pada satu bulan.
func (s *Service) LastTransactions(ctx context.Context, month time.Time, limit int32) ([]Transaction, error) {
	rows, err := s.q.ListTransactions(ctx, store.ListTransactionsParams{
		MonthStart: dateOf(month),
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		result = append(result, s.transactionFromListRow(row))
	}
	return result, nil
}

func (s *Service) transactionFromStore(row store.Transaction, category *store.Category, account *store.Account) Transaction {
	tx := Transaction{
		ID:         row.ID,
		Type:       row.Type,
		Amount:     row.Amount,
		Note:       row.Note,
		CategoryID: int8Value(row.CategoryID),
		AccountID:  int8Value(row.AccountID),
		OccurredAt: row.OccurredAt.Time.In(s.loc),
		Month:      FormatMonth(pgDateToTime(row.MonthStart, s.loc)),
		Source:     row.Source,
	}
	if category != nil {
		tx.CategoryName = category.Name
	}
	if account != nil {
		tx.AccountName = account.Name
	}
	return tx
}

func (s *Service) transactionFromGetRow(row store.GetTransactionRow) Transaction {
	return s.buildTransaction(
		row.ID, row.Type, row.Amount, row.Note,
		row.CategoryID, row.CategoryName, row.AccountID, row.AccountName,
		row.OccurredAt, row.MonthStart, row.Source,
	)
}

func (s *Service) transactionFromListRow(row store.ListTransactionsRow) Transaction {
	return s.buildTransaction(
		row.ID, row.Type, row.Amount, row.Note,
		row.CategoryID, row.CategoryName, row.AccountID, row.AccountName,
		row.OccurredAt, row.MonthStart, row.Source,
	)
}

func (s *Service) transactionFromLastRow(row store.LastTransactionRow) Transaction {
	return s.buildTransaction(
		row.ID, row.Type, row.Amount, row.Note,
		row.CategoryID, row.CategoryName, row.AccountID, row.AccountName,
		row.OccurredAt, row.MonthStart, row.Source,
	)
}

func (s *Service) buildTransaction(
	id int64,
	txType string,
	amount int64,
	note string,
	categoryID pgtype.Int8,
	categoryName pgtype.Text,
	accountID pgtype.Int8,
	accountName pgtype.Text,
	occurredAt pgtype.Timestamptz,
	monthStart pgtype.Date,
	source string,
) Transaction {
	return Transaction{
		ID:           id,
		Type:         txType,
		Amount:       amount,
		Note:         note,
		CategoryID:   int8Value(categoryID),
		CategoryName: textValueString(categoryName),
		AccountID:    int8Value(accountID),
		AccountName:  textValueString(accountName),
		OccurredAt:   occurredAt.Time.In(s.loc),
		Month:        FormatMonth(pgDateToTime(monthStart, s.loc)),
		Source:       source,
	}
}

func categoryIDPtr(c *store.Category) *int64 {
	if c == nil {
		return nil
	}
	id := c.ID
	return &id
}

func accountIDPtr(a *store.Account) *int64 {
	if a == nil {
		return nil
	}
	id := a.ID
	return &id
}

func trimPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func textValueString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
