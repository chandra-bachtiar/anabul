package tools

import "time"

// Transaction already carries the category and account names so the UI does not
// have to join anything.
type Transaction struct {
	ID           int64     `json:"id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	Note         string    `json:"note"`
	CategoryID   *int64    `json:"category_id"`
	CategoryName string    `json:"category_name,omitempty"`
	AccountID    *int64    `json:"account_id"`
	AccountName  string    `json:"account_name,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	Month        string    `json:"month"`
	Source       string    `json:"source"`
}

type Category struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

type Account struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type BudgetRow struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Amount       int64  `json:"amount"`
	Spent        int64  `json:"spent"`
	Remaining    int64  `json:"remaining"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
}

type NeedRow struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Amount        int64  `json:"amount"`
	DueDate       string `json:"due_date"`
	Status        string `json:"status"`
	CategoryID    *int64 `json:"category_id"`
	CategoryName  string `json:"category_name,omitempty"`
	TransactionID *int64 `json:"transaction_id"`
	Overdue       bool   `json:"overdue"`
	DueSoon       bool   `json:"due_soon"`
}

// RecurringNeedRow is a monthly need template.
type RecurringNeedRow struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Amount       int64  `json:"amount"`
	DueDay       int    `json:"due_day"`
	CategoryID   *int64 `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
	Active       bool   `json:"active"`
}

type DebtPaymentRow struct {
	ID        int64     `json:"id"`
	Amount    int64     `json:"amount"`
	PaidAt    time.Time `json:"paid_at"`
	Direction string    `json:"-"`
}

// DebtRow is a debt or receivable with its remaining amount and payment history.
type DebtRow struct {
	ID            int64            `json:"id"`
	Person        string           `json:"person"`
	Direction     string           `json:"direction"`
	AmountTotal   int64            `json:"amount_total"`
	Paid          int64            `json:"paid"`
	Remaining     int64            `json:"remaining"`
	Note          string           `json:"note"`
	DueDate       *string          `json:"due_date"`
	Status        string           `json:"status"`
	Overdue       bool             `json:"overdue"`
	CreatedAt     time.Time        `json:"created_at"`
	PaymentCount  int64            `json:"payment_count"`
	LastPaymentAt *time.Time       `json:"last_payment_at"`
	Payments      []DebtPaymentRow `json:"payments,omitempty"`
}

// DebtTotals keeps the product's own wording for the two directions, since the
// UI labels come straight from these values.
type DebtTotals struct {
	Hutang  int64 `json:"hutang"`
	Piutang int64 `json:"piutang"`
}

type CategorySpend struct {
	CategoryID   *int64 `json:"category_id"`
	CategoryName string `json:"category_name"`
	Spent        int64  `json:"spent"`
	Budget       int64  `json:"budget"`
	Remaining    int64  `json:"remaining"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
}

type Summary struct {
	Month        string          `json:"month"`
	MonthLabel   string          `json:"month_label"`
	Locked       bool            `json:"locked"`
	Income       int64           `json:"income"`
	Expense      int64           `json:"expense"`
	Net          int64           `json:"net"`
	BudgetTotal  int64           `json:"budget_total"`
	BudgetSpent  int64           `json:"budget_spent"`
	BudgetLeft   int64           `json:"budget_remaining"`
	Budgets      []BudgetRow     `json:"budgets"`
	Categories   []CategorySpend `json:"categories"`
	Profitable   bool            `json:"profitable"`
	UnpaidNeeds  int64           `json:"unpaid_needs"`
	NeedsTotal   int64           `json:"unpaid_needs_amount"`
	DebtTotals   DebtTotals      `json:"debts"`
	TopCategory  string          `json:"top_category"`
	TopSpent     int64           `json:"top_category_amount"`
	BiggestDrain []CategorySpend `json:"biggest_drain"`

	transactionCount int64
}

type Dashboard struct {
	Month            string          `json:"month"`
	MonthLabel       string          `json:"month_label"`
	Locked           bool            `json:"locked"`
	Mood             string          `json:"mood"`
	Income           int64           `json:"income"`
	Expense          int64           `json:"expense"`
	Net              int64           `json:"net"`
	BudgetTotal      int64           `json:"budget_total"`
	BudgetSpent      int64           `json:"budget_spent"`
	BudgetRemaining  int64           `json:"budget_remaining"`
	BudgetStatus     string          `json:"budget_status"`
	Budgets          []BudgetRow     `json:"budgets"`
	TopCategories    []CategorySpend `json:"top_categories"`
	Needs            []NeedRow       `json:"needs"`
	NeedsUnpaid      int64           `json:"needs_unpaid"`
	NeedsUnpaidTotal int64           `json:"needs_unpaid_amount"`
	Debts            DebtTotals      `json:"debts"`
	Recent           []Transaction   `json:"recent"`
	TransactionCount int64           `json:"transaction_count"`
}

type MonthPreview struct {
	Summary
	UnpaidNeedRows []NeedRow   `json:"unpaid_need_rows"`
	OpenDebts      []DebtRow   `json:"open_debts"`
	DueDebts       []DebtRow   `json:"due_debts"`
	Snapshot       Snapshot    `json:"snapshot"`
	Warnings       []string    `json:"warnings"`
	BudgetOver     []BudgetRow `json:"budget_over"`
}

// Snapshot is what lands in month_closings.snapshot.
type Snapshot struct {
	Month            string          `json:"month"`
	MonthLabel       string          `json:"month_label"`
	ClosedAt         time.Time       `json:"closed_at"`
	Income           int64           `json:"income"`
	Expense          int64           `json:"expense"`
	Net              int64           `json:"net"`
	BudgetTotal      int64           `json:"budget_total"`
	BudgetSpent      int64           `json:"budget_spent"`
	Budgets          []BudgetRow     `json:"budgets"`
	Categories       []CategorySpend `json:"categories"`
	UnpaidNeeds      []NeedRow       `json:"unpaid_needs"`
	DebtTotals       DebtTotals      `json:"debts"`
	TransactionCount int64           `json:"transaction_count"`
	CarriedBudgets   int             `json:"carried_budgets"`
	CarriedNeeds     int             `json:"carried_needs"`
}
