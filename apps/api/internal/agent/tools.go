package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"anabul/internal/tools"
)

type toolSchema struct {
	Type     string             `json:"type"`
	Function toolSchemaFunction `json:"function"`
}

type toolSchemaFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// toolDef pairs the schema the model sees with the call that executes it.
type toolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
	Risky       bool
	Run         func(ctx context.Context, args json.RawMessage) (any, error)
}

func schema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	if required == nil {
		required = []string{}
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func obj(description string) map[string]any {
	return map[string]any{"type": "object", "description": description}
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func strEnum(description string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": description, "enum": values}
}

func integer(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolean(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

const amountHint = "integer rupiah, no separators, e.g. 25000"

// registry lists every tool the model may call.
func (a *Agent) registry() []toolDef {
	return []toolDef{
		{
			Name:        "add_transaction",
			Description: "Record a new expense or income. Fill date when the user names another day, e.g. kemarin or a specific date.",
			Parameters: schema(map[string]any{
				"amount":   integer(amountHint),
				"note":     str("short note, e.g. kopi"),
				"type":     strEnum("expense for money going out, income for money coming in", "expense", "income"),
				"category": str("category name (optional)"),
				"account":  str("account name: cash, bank, e-wallet (optional)"),
				"date":     str("entry date as YYYY-MM-DD when it is not today, e.g. kemarin means yesterday's date"),
			}, "amount"),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Amount   json.Number `json:"amount"`
					Note     string      `json:"note"`
					Type     string      `json:"type"`
					Category string      `json:"category"`
					Account  string      `json:"account"`
					Date     string      `json:"date"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				amount, err := amountOf(in.Amount)
				if err != nil {
					return nil, err
				}
				occurredAt, err := parseDate(in.Date, a.tools.Location())
				if err != nil {
					return nil, err
				}
				tx, err := a.tools.AddTransaction(ctx, tools.AddTransactionInput{
					Type:       in.Type,
					Amount:     amount,
					Note:       in.Note,
					Category:   in.Category,
					Account:    in.Account,
					OccurredAt: occurredAt,
					Source:     channelFrom(ctx),
				})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":       true,
					"id":       tx.ID,
					"amount":   tx.Amount,
					"note":     tx.Note,
					"type":     tx.Type,
					"category": tx.CategoryName,
					"month":    tx.Month,
					"date":     tx.OccurredAt.Format("2006-01-02"),
				}, nil
			},
		},
		{
			Name:        "update_transaction",
			Description: "Change an existing transaction, e.g. to fix 'yang tadi salah'. Use last=true for the most recent one.",
			Parameters: schema(map[string]any{
				"id":       integer("transaction id when known"),
				"last":     boolean("true when the target is the latest transaction"),
				"amount":   integer(amountHint),
				"note":     str("new note"),
				"type":     strEnum("", "expense", "income"),
				"category": str("new category name"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					ID       *int64      `json:"id"`
					Last     bool        `json:"last"`
					Amount   json.Number `json:"amount"`
					Note     *string     `json:"note"`
					Type     *string     `json:"type"`
					Category *string     `json:"category"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				target := int64(0)
				if in.ID != nil {
					target = *in.ID
				} else if in.Last {
					last, err := a.tools.LastTransaction(ctx)
					if err != nil {
						return nil, err
					}
					target = last.ID
				}
				if target == 0 {
					return nil, tools.NewInvalidError("sebutkan id transaksi atau pakai last=true")
				}

				patch := tools.UpdateTransactionInput{ID: target, Note: in.Note, Type: in.Type}
				if in.Amount != "" {
					amount, err := amountOf(in.Amount)
					if err != nil {
						return nil, err
					}
					patch.Amount = &amount
				}
				if in.Category != nil {
					categoryID, err := a.categoryID(ctx, *in.Category)
					if err != nil {
						return nil, err
					}
					patch.CategoryID = &categoryID
				}
				tx, err := a.tools.UpdateTransaction(ctx, patch)
				if err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "id": tx.ID, "amount": tx.Amount, "note": tx.Note}, nil
			},
		},
		{
			Name:        "delete_transaction",
			Description: "Delete a transaction. Risky action: always ask the user to confirm first.",
			Parameters: schema(map[string]any{
				"id":   integer("transaction id"),
				"last": boolean("true when the target is the latest transaction"),
			}),
			Risky: true,
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					ID   *int64 `json:"id"`
					Last bool   `json:"last"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				target := int64(0)
				if in.ID != nil {
					target = *in.ID
				} else if in.Last {
					last, err := a.tools.LastTransaction(ctx)
					if err != nil {
						return nil, err
					}
					target = last.ID
				}
				if target == 0 {
					return nil, tools.NewInvalidError("sebutkan id transaksi atau pakai last=true")
				}
				if err := a.tools.DeleteTransaction(ctx, target); err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "id": target, "deleted": true}, nil
			},
		},
		{
			Name:        "get_summary",
			Description: "Income, expense, category, and budget summary for one month.",
			Parameters: schema(map[string]any{
				"month":    str("month as YYYY-MM, empty for the current month"),
				"group_by": strEnum("", "category"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Month   string `json:"month"`
					GroupBy string `json:"group_by"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				summary, err := a.tools.MonthSummary(ctx, in.Month)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":               true,
					"month":            summary.Month,
					"income":           summary.Income,
					"expense":          summary.Expense,
					"net":              summary.Net,
					"budget_total":     summary.BudgetTotal,
					"budget_remaining": summary.BudgetLeft,
					"categories":       summary.Categories,
					"biggest_drain":    summary.BiggestDrain,
				}, nil
			},
		},
		{
			Name:        "set_budget",
			Description: "Store the budget amount for one category in one month.",
			Parameters: schema(map[string]any{
				"category": str("category name"),
				"amount":   integer(amountHint),
				"month":    str("month as YYYY-MM, empty for the current month"),
			}, "category", "amount"),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Category string      `json:"category"`
					Amount   json.Number `json:"amount"`
					Month    string      `json:"month"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				amount, err := amountOf(in.Amount)
				if err != nil {
					return nil, err
				}
				categoryID, err := a.categoryID(ctx, in.Category)
				if err != nil {
					return nil, err
				}
				budget, err := a.tools.SetBudget(ctx, in.Month, categoryID, amount)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":        true,
					"category":  budget.CategoryName,
					"amount":    budget.Amount,
					"spent":     budget.Spent,
					"remaining": budget.Remaining,
				}, nil
			},
		},
		{
			Name:        "get_budget_remaining",
			Description: "Remaining budget per category for one month.",
			Parameters: schema(map[string]any{
				"category": str("category name, empty for all of them"),
				"month":    str("month as YYYY-MM, empty for the current month"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Category string `json:"category"`
					Month    string `json:"month"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				rows, err := a.tools.ListBudgets(ctx, in.Month)
				if err != nil {
					return nil, err
				}
				if strings.TrimSpace(in.Category) != "" {
					filtered := make([]tools.BudgetRow, 0, 1)
					for _, row := range rows {
						if strings.EqualFold(row.CategoryName, strings.TrimSpace(in.Category)) {
							filtered = append(filtered, row)
						}
					}
					if len(filtered) == 0 {
						return nil, tools.NewNotFoundError("kategori %s belum punya budget bulan ini", in.Category)
					}
					rows = filtered
				}
				total := int64(0)
				remaining := int64(0)
				for _, row := range rows {
					total += row.Amount
					remaining += row.Remaining
				}
				return map[string]any{"ok": true, "budgets": rows, "total": total, "remaining": remaining}, nil
			},
		},
		{
			Name:        "add_debt",
			Description: "Record a debt (the user owes) or a receivable (someone owes the user).",
			Parameters: schema(map[string]any{
				"person":    str("person's name"),
				"direction": strEnum("hutang when the user owes, piutang when the other person does", "hutang", "piutang"),
				"amount":    integer(amountHint),
				"note":      str("note (optional)"),
				"due_date":  str("due date as YYYY-MM-DD (optional)"),
			}, "person", "direction", "amount"),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Person    string      `json:"person"`
					Direction string      `json:"direction"`
					Amount    json.Number `json:"amount"`
					Note      string      `json:"note"`
					DueDate   string      `json:"due_date"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				amount, err := amountOf(in.Amount)
				if err != nil {
					return nil, err
				}
				dueDate, err := parseDate(in.DueDate, a.tools.Location())
				if err != nil {
					return nil, err
				}
				debt, err := a.tools.AddDebt(ctx, tools.AddDebtInput{
					Person:    in.Person,
					Direction: in.Direction,
					Amount:    amount,
					Note:      in.Note,
					DueDate:   dueDate,
				})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":        true,
					"id":        debt.ID,
					"person":    debt.Person,
					"direction": debt.Direction,
					"amount":    debt.AmountTotal,
					"remaining": debt.Remaining,
				}, nil
			},
		},
		{
			Name:        "add_debt_payment",
			Description: "Record a payment or installment on a debt or receivable. The payment also books a transaction (expense for hutang, income for piutang) unless record_transaction is false.",
			Parameters: schema(map[string]any{
				"person":             str("person's name"),
				"debt_id":            integer("debt id when known"),
				"amount":             integer(amountHint),
				"date":               str("payment date as YYYY-MM-DD when it is not today"),
				"record_transaction": boolean("false to keep the payment out of the daily ledger (default true)"),
			}, "amount"),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Person            string      `json:"person"`
					DebtID            *int64      `json:"debt_id"`
					Amount            json.Number `json:"amount"`
					Date              string      `json:"date"`
					RecordTransaction *bool       `json:"record_transaction"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				amount, err := amountOf(in.Amount)
				if err != nil {
					return nil, err
				}
				paidAt, err := parseDate(in.Date, a.tools.Location())
				if err != nil {
					return nil, err
				}
				record := true
				if in.RecordTransaction != nil {
					record = *in.RecordTransaction
				}
				debt, err := a.tools.AddDebtPayment(ctx, tools.AddDebtPaymentInput{
					Person:            in.Person,
					DebtID:            in.DebtID,
					Amount:            amount,
					PaidAt:            paidAt,
					RecordTransaction: record,
				})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":                 true,
					"id":                 debt.ID,
					"person":             debt.Person,
					"direction":          debt.Direction,
					"paid":               debt.Paid,
					"remaining":          debt.Remaining,
					"status":             debt.Status,
					"payment_count":      debt.PaymentCount,
					"last_payment":       debt.LastPaymentAt,
					"record_transaction": record,
				}, nil
			},
		},
		{
			Name:        "list_debts",
			Description: "List debts and receivables with their remaining amount and due date.",
			Parameters: schema(map[string]any{
				"status": strEnum("open or settled, empty for all", "open", "settled"),
				"person": str("filter by person"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Status string `json:"status"`
					Person string `json:"person"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				rows, err := a.tools.ListDebts(ctx, in.Status, in.Person)
				if err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "debts": rows}, nil
			},
		},
		{
			Name:        "list_monthly_needs",
			Description: "List monthly needs (electricity, internet, and the like).",
			Parameters: schema(map[string]any{
				"month":  str("month as YYYY-MM, empty for the current month"),
				"status": strEnum("unpaid or paid, empty for all", "unpaid", "paid"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Month  string `json:"month"`
					Status string `json:"status"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				rows, err := a.tools.ListMonthlyNeeds(ctx, in.Month, in.Status)
				if err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "needs": rows}, nil
			},
		},
		{
			Name:        "mark_need_paid",
			Description: "Mark a monthly need as paid; books the matching expense.",
			Parameters: schema(map[string]any{
				"name":   str("need name, e.g. listrik"),
				"amount": integer("amount when it differs from the estimate (optional)"),
			}, "name"),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Name   string      `json:"name"`
					Amount json.Number `json:"amount"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				month := a.tools.CurrentMonth()
				need, err := a.tools.FindNeedByName(ctx, month, in.Name)
				if err != nil {
					return nil, err
				}
				var amount *int64
				if in.Amount != "" {
					value, err := amountOf(in.Amount)
					if err != nil {
						return nil, err
					}
					amount = &value
				}
				paid, err := a.tools.MarkNeedPaid(ctx, need.ID, amount)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":             true,
					"id":             paid.ID,
					"name":           paid.Name,
					"amount":         paid.Amount,
					"status":         paid.Status,
					"due_date":       paid.DueDate,
					"transaction_id": paid.TransactionID,
				}, nil
			},
		},
		{
			Name:        "preview_close_month",
			Description: "Summary before closing the month: totals, budget, unpaid needs, debts.",
			Parameters: schema(map[string]any{
				"month": str("month as YYYY-MM, empty for the current month"),
			}),
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Month string `json:"month"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				preview, err := a.tools.MonthPreview(ctx, in.Month)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":             true,
					"month":          preview.Month,
					"income":         preview.Income,
					"expense":        preview.Expense,
					"net":            preview.Net,
					"budget_total":   preview.BudgetTotal,
					"budget_left":    preview.BudgetLeft,
					"unpaid_needs":   preview.UnpaidNeedRows,
					"debts":          preview.DebtTotals,
					"warnings":       preview.Warnings,
					"already_locked": preview.Locked,
				}, nil
			},
		},
		{
			Name:        "close_month",
			Description: "Close the month: lock its transactions, store the snapshot, prepare the next month. Risky action, always needs the user's confirmation.",
			Parameters: schema(map[string]any{
				"month": str("month as YYYY-MM to close, empty for the current month"),
			}),
			Risky: true,
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Month string `json:"month"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				snapshot, err := a.tools.CloseMonth(ctx, in.Month)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"ok":              true,
					"month":           snapshot.Month,
					"income":          snapshot.Income,
					"expense":         snapshot.Expense,
					"net":             snapshot.Net,
					"carried_budgets": snapshot.CarriedBudgets,
					"carried_needs":   snapshot.CarriedNeeds,
				}, nil
			},
		},
		{
			Name:        "reopen_month",
			Description: "Reopen a closed month so it can be edited again. Risky action, needs the user's confirmation.",
			Parameters: schema(map[string]any{
				"month": str("month as YYYY-MM to reopen"),
			}, "month"),
			Risky: true,
			Run: func(ctx context.Context, args json.RawMessage) (any, error) {
				var in struct {
					Month string `json:"month"`
				}
				if err := decodeArgs(args, &in); err != nil {
					return nil, err
				}
				if err := a.tools.ReopenMonth(ctx, in.Month); err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "month": in.Month, "reopened": true}, nil
			},
		},
	}
}

// runTool executes one tool; risky ones only ever run through the confirmation flow.
func (a *Agent) runTool(ctx context.Context, name string, args json.RawMessage, confirmed bool) (string, error) {
	def, found := a.toolByName(name)
	if !found {
		return "", fmt.Errorf("unknown tool %s", name)
	}

	if def.Risky && !confirmed {
		confirmation, err := a.createPending(ctx, def, args)
		if err != nil {
			return "", err
		}
		encoded, err := json.Marshal(confirmation)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}

	result, err := def.Run(ctx, args)
	if err != nil {
		return encodeToolError(err), nil
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (a *Agent) toolByName(name string) (toolDef, bool) {
	for _, def := range a.registry() {
		if def.Name == name {
			return def, true
		}
	}
	return toolDef{}, false
}

func (a *Agent) toolSchemas() []toolSchema {
	defs := a.registry()
	schemas := make([]toolSchema, 0, len(defs))
	for _, def := range defs {
		schemas = append(schemas, toolSchema{
			Type: "function",
			Function: toolSchemaFunction{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}
	return schemas
}

func decodeArgs(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return tools.NewInvalidError("argumen tool tidak valid: %s", err.Error())
	}
	return nil
}

// amountOf takes whatever the model sent: a number or a string like "25rb".
func amountOf(value json.Number) (int64, error) {
	raw := strings.TrimSpace(value.String())
	if raw == "" {
		return 0, tools.NewInvalidError("nominalnya belum ada")
	}
	if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if parsed <= 0 {
			return 0, tools.NewInvalidError("nominal harus lebih dari 0")
		}
		return parsed, nil
	}
	return tools.ParseAmount(raw)
}

func parseDate(raw string, loc *time.Location) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return nil, tools.NewInvalidError("format tanggal harus YYYY-MM-DD")
	}
	return &parsed, nil
}

func (a *Agent) categoryID(ctx context.Context, name string) (int64, error) {
	categories, err := a.tools.ListCategories(ctx)
	if err != nil {
		return 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(name))
	for _, category := range categories {
		if strings.ToLower(category.Name) == needle {
			return category.ID, nil
		}
	}
	names := make([]string, 0, len(categories))
	for _, category := range categories {
		names = append(names, category.Name)
	}
	return 0, tools.NewInvalidError("kategori %q tidak ada", name).WithAvailable(names...)
}

// encodeToolError hands the model a JSON error it can relay to the user.
func encodeToolError(err error) string {
	var domainErr *tools.Error
	if errors.As(err, &domainErr) {
		encoded, marshalErr := json.Marshal(domainErr)
		if marshalErr != nil {
			return `{"error":"ada gangguan di tool"}`
		}
		return string(encoded)
	}
	return `{"error":"ada gangguan saat menjalankan aksi"}`
}
