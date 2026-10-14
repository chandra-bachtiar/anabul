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

// AddDebtInput adalah parameter pencatatan hutang/piutang baru.
type AddDebtInput struct {
	Person    string
	Direction string
	Amount    int64
	Note      string
	DueDate   *time.Time
}

// AddDebtPaymentInput adalah parameter pembayaran hutang/piutang.
type AddDebtPaymentInput struct {
	Person        string
	DebtID        *int64
	Amount        int64
	PaidAt        *time.Time
	TransactionID *int64
}

// AddDebt mencatat hutang (user berhutang) atau piutang (orang berhutang ke user).
func (s *Service) AddDebt(ctx context.Context, in AddDebtInput) (DebtRow, error) {
	person := strings.TrimSpace(in.Person)
	if person == "" {
		return DebtRow{}, invalid("nama orangnya belum ada")
	}
	direction, err := normalizeDirection(in.Direction)
	if err != nil {
		return DebtRow{}, err
	}
	if in.Amount <= 0 {
		return DebtRow{}, invalid("nominal harus lebih dari 0")
	}

	row, err := s.q.CreateDebt(ctx, store.CreateDebtParams{
		Person:      person,
		Direction:   direction,
		AmountTotal: in.Amount,
		Note:        strings.TrimSpace(in.Note),
		DueDate:     dateOfPtr(in.DueDate),
	})
	if err != nil {
		return DebtRow{}, err
	}
	return s.debtRow(row.ID, person, direction, row.AmountTotal, row.Note, row.DueDate, row.Status, 0, s.Now()), nil
}

// AddDebtPayment mencatat pembayaran/cicilan dan menandai lunas bila sisa 0.
func (s *Service) AddDebtPayment(ctx context.Context, in AddDebtPaymentInput) (DebtRow, error) {
	if in.Amount <= 0 {
		return DebtRow{}, invalid("nominal pembayaran harus lebih dari 0")
	}

	debt, err := s.pickDebt(ctx, in)
	if err != nil {
		return DebtRow{}, err
	}

	paidBefore, err := s.q.DebtPaidTotal(ctx, debt.ID)
	if err != nil {
		return DebtRow{}, err
	}
	remaining := debt.AmountTotal - paidBefore
	if in.Amount > remaining {
		return DebtRow{}, invalid("pembayaran %s melebihi sisa %s", FormatAmount(in.Amount), FormatAmount(remaining))
	}

	paidAt := s.Now()
	if in.PaidAt != nil {
		paidAt = in.PaidAt.In(s.loc)
	}

	err = s.InTx(ctx, func(q *store.Queries) error {
		if _, err := q.CreateDebtPayment(ctx, store.CreateDebtPaymentParams{
			DebtID:        debt.ID,
			Amount:        in.Amount,
			PaidAt:        pgtype.Timestamptz{Time: paidAt, Valid: true},
			TransactionID: int8Ptr(in.TransactionID),
		}); err != nil {
			return err
		}
		status := "open"
		if in.Amount == remaining {
			status = "settled"
		}
		return q.SetDebtStatus(ctx, store.SetDebtStatusParams{ID: debt.ID, Status: status})
	})
	if err != nil {
		return DebtRow{}, err
	}

	detail, err := s.q.GetDebt(ctx, debt.ID)
	if err != nil {
		return DebtRow{}, err
	}
	return s.debtRowFromDetail(detail, nil), nil
}

// ListDebts mengembalikan hutang piutang beserta sisa dan statusnya.
func (s *Service) ListDebts(ctx context.Context, status, person string) ([]DebtRow, error) {
	if status != "" && status != "open" && status != "settled" {
		return nil, invalid("status harus 'open' atau 'settled'")
	}
	rows, err := s.q.ListDebts(ctx, store.ListDebtsParams{
		Status: textPtr(trimPtr(status)),
		Person: textPtr(trimPtr(person)),
	})
	if err != nil {
		return nil, err
	}
	today := StartOfDay(s.Now(), s.loc)
	result := make([]DebtRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, DebtRow{
			ID:          row.ID,
			Person:      row.Person,
			Direction:   row.Direction,
			AmountTotal: row.AmountTotal,
			Paid:        row.Paid,
			Remaining:   row.Remaining,
			Note:        row.Note,
			DueDate:     formatDate(dateValue(row.DueDate)),
			Status:      row.Status,
			Overdue:     isOverdue(row.DueDate, row.Status, today),
			CreatedAt:   row.CreatedAt.Time.In(s.loc),
		})
	}
	return result, nil
}

// GetDebt mengambil satu hutang lengkap dengan riwayat pembayarannya.
func (s *Service) GetDebt(ctx context.Context, id int64) (DebtRow, error) {
	detail, err := s.q.GetDebt(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DebtRow{}, notFound("hutang/piutang tidak ditemukan")
		}
		return DebtRow{}, err
	}
	payments, err := s.q.ListDebtPayments(ctx, id)
	if err != nil {
		return DebtRow{}, err
	}
	history := make([]DebtPaymentRow, 0, len(payments))
	for _, p := range payments {
		history = append(history, DebtPaymentRow{ID: p.ID, Amount: p.Amount, PaidAt: p.PaidAt.Time.In(s.loc)})
	}
	return s.debtRowFromDetail(detail, history), nil
}

// DebtTotals adalah total sisa hutang dan piutang yang masih berjalan.
func (s *Service) DebtTotals(ctx context.Context) (DebtTotals, error) {
	row, err := s.q.DebtTotals(ctx)
	if err != nil {
		return DebtTotals{}, err
	}
	return DebtTotals{Hutang: row.Hutang, Piutang: row.Piutang}, nil
}

type debtRecord struct {
	ID          int64
	AmountTotal int64
	Direction   string
}

func (s *Service) pickDebt(ctx context.Context, in AddDebtPaymentInput) (debtRecord, error) {
	if in.DebtID != nil {
		detail, err := s.q.GetDebt(ctx, *in.DebtID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return debtRecord{}, notFound("hutang/piutang tidak ditemukan")
			}
			return debtRecord{}, err
		}
		if detail.Status == "settled" {
			return debtRecord{}, &Error{Code: CodeConflict, Message: "hutang/piutang " + detail.Person + " sudah lunas"}
		}
		return debtRecord{ID: detail.ID, AmountTotal: detail.AmountTotal, Direction: detail.Direction}, nil
	}

	person := strings.TrimSpace(in.Person)
	if person == "" {
		return debtRecord{}, invalid("sebutkan nama orangnya atau id hutangnya")
	}
	candidates, err := s.q.FindOpenDebtByPerson(ctx, store.FindOpenDebtByPersonParams{Lower: person})
	if err != nil {
		return debtRecord{}, err
	}
	switch len(candidates) {
	case 0:
		return debtRecord{}, notFound("tidak ada hutang/piutang berjalan untuk %s", person)
	case 1:
		return debtRecord{ID: candidates[0].ID, AmountTotal: candidates[0].AmountTotal, Direction: candidates[0].Direction}, nil
	}

	options := make([]string, 0, len(candidates))
	for _, c := range candidates {
		options = append(options, directionLabel(c.Direction)+" "+FormatAmount(c.Remaining)+" ("+c.Note+")")
	}
	return debtRecord{}, &Error{
		Code:      CodeAmbiguous,
		Message:   "ada beberapa catatan untuk " + person + ", pilih satu dulu ya",
		Available: options,
	}
}

func (s *Service) debtRow(id int64, person, direction string, amountTotal int64, note string,
	dueDate pgtype.Date, status string, paid int64, now time.Time) DebtRow {
	return DebtRow{
		ID:          id,
		Person:      person,
		Direction:   direction,
		AmountTotal: amountTotal,
		Paid:        paid,
		Remaining:   amountTotal - paid,
		Note:        note,
		DueDate:     formatDate(dateValue(dueDate)),
		Status:      status,
		Overdue:     isOverdue(dueDate, status, StartOfDay(now, s.loc)),
		CreatedAt:   now.In(s.loc),
	}
}

func (s *Service) debtRowFromDetail(detail store.GetDebtRow, payments []DebtPaymentRow) DebtRow {
	row := DebtRow{
		ID:          detail.ID,
		Person:      detail.Person,
		Direction:   detail.Direction,
		AmountTotal: detail.AmountTotal,
		Paid:        detail.Paid,
		Remaining:   detail.Remaining,
		Note:        detail.Note,
		DueDate:     formatDate(dateValue(detail.DueDate)),
		Status:      detail.Status,
		Overdue:     isOverdue(detail.DueDate, detail.Status, StartOfDay(s.Now(), s.loc)),
		CreatedAt:   detail.CreatedAt.Time.In(s.loc),
		Payments:    payments,
	}
	return row
}

func normalizeDirection(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "hutang", "utang", "debt", "i_owe":
		return "hutang", nil
	case "piutang", "receivable", "owed_to_me":
		return "piutang", nil
	case "":
		return "", invalid("arahnya harus 'hutang' atau 'piutang'")
	default:
		return "", invalid("arah %q tidak dikenal, pakai 'hutang' atau 'piutang'", raw)
	}
}

func directionLabel(direction string) string {
	if direction == "piutang" {
		return "piutang"
	}
	return "hutang"
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	value := t.Format("2006-01-02")
	return &value
}

func isOverdue(dueDate pgtype.Date, status string, today time.Time) bool {
	if status != "open" || !dueDate.Valid {
		return false
	}
	due := time.Date(dueDate.Time.Year(), dueDate.Time.Month(), dueDate.Time.Day(), 0, 0, 0, 0, today.Location())
	return due.Before(today)
}
