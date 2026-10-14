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

// toolDef menggabungkan skema tool dengan eksekusinya di layer tools.
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

const amountHint = "nominal integer rupiah tanpa titik/koma, mis. 25000"

// registry berisi semua tool yang boleh dipanggil model.
func (a *Agent) registry() []toolDef {
	return []toolDef{
		{
			Name:        "add_transaction",
			Description: "Catat pengeluaran atau pemasukan baru. Isi date kalau pengguna menyebut waktu lain, misalnya kemarin atau tanggal tertentu.",
			Parameters: schema(map[string]any{
				"amount":   integer(amountHint),
				"note":     str("catatan singkat, mis. kopi"),
				"type":     strEnum("expense untuk pengeluaran, income untuk pemasukan", "expense", "income"),
				"category": str("nama kategori (opsional)"),
				"account":  str("nama akun: cash, bank, e-wallet (opsional)"),
				"date":     str("tanggal catatan format YYYY-MM-DD kalau bukan hari ini, mis. kemarin berarti tanggal kemarin"),
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
			Description: "Ubah transaksi, mis. untuk koreksi 'yang tadi salah'. Pakai last=true untuk transaksi terakhir.",
			Parameters: schema(map[string]any{
				"id":       integer("id transaksi kalau diketahui"),
				"last":     boolean("true kalau yang dimaksud transaksi terakhir"),
				"amount":   integer(amountHint),
				"note":     str("catatan baru"),
				"type":     strEnum("", "expense", "income"),
				"category": str("nama kategori baru"),
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
			Description: "Hapus transaksi. Aksi berisiko: selalu minta konfirmasi ke pengguna.",
			Parameters: schema(map[string]any{
				"id":   integer("id transaksi"),
				"last": boolean("true kalau yang dimaksud transaksi terakhir"),
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
			Description: "Ringkasan pemasukan, pengeluaran, kategori, dan budget pada satu bulan.",
			Parameters: schema(map[string]any{
				"month":    str("bulan format YYYY-MM, kosongkan untuk bulan ini"),
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
			Description: "Simpan nominal budget satu kategori pada satu bulan.",
			Parameters: schema(map[string]any{
				"category": str("nama kategori"),
				"amount":   integer(amountHint),
				"month":    str("bulan YYYY-MM, kosongkan untuk bulan ini"),
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
			Description: "Sisa budget per kategori pada satu bulan.",
			Parameters: schema(map[string]any{
				"category": str("nama kategori, kosongkan untuk semua"),
				"month":    str("bulan YYYY-MM, kosongkan untuk bulan ini"),
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
			Description: "Catat hutang (user berhutang) atau piutang (orang berhutang ke user).",
			Parameters: schema(map[string]any{
				"person":    str("nama orang"),
				"direction": strEnum("hutang kalau user yang berhutang, piutang kalau orang lain", "hutang", "piutang"),
				"amount":    integer(amountHint),
				"note":      str("catatan (opsional)"),
				"due_date":  str("jatuh tempo YYYY-MM-DD (opsional)"),
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
			Description: "Catat pembayaran/cicilan hutang atau piutang. Pembayaran otomatis dibuatkan transaksi (pengeluaran untuk hutang, pemasukan untuk piutang) kecuali record_transaction diisi false.",
			Parameters: schema(map[string]any{
				"person":             str("nama orang"),
				"debt_id":            integer("id hutang kalau diketahui"),
				"amount":             integer(amountHint),
				"date":               str("tanggal pembayaran format YYYY-MM-DD kalau bukan hari ini"),
				"record_transaction": boolean("false kalau pembayaran tidak boleh masuk catatan harian (bawaan true)"),
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
				catat := true
				if in.RecordTransaction != nil {
					catat = *in.RecordTransaction
				}
				debt, err := a.tools.AddDebtPayment(ctx, tools.AddDebtPaymentInput{
					Person:         in.Person,
					DebtID:         in.DebtID,
					Amount:         amount,
					PaidAt:         paidAt,
					CatatTransaksi: catat,
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
					"record_transaction": catat,
				}, nil
			},
		},
		{
			Name:        "list_debts",
			Description: "Daftar hutang piutang beserta sisa dan jatuh temponya.",
			Parameters: schema(map[string]any{
				"status": strEnum("open atau settled, kosongkan untuk semua", "open", "settled"),
				"person": str("filter nama orang"),
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
			Description: "Daftar kebutuhan bulanan (listrik, internet, dan sejenisnya).",
			Parameters: schema(map[string]any{
				"month":  str("bulan YYYY-MM, kosongkan untuk bulan ini"),
				"status": strEnum("unpaid atau paid, kosongkan untuk semua", "unpaid", "paid"),
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
			Description: "Tandai kebutuhan bulanan sudah dibayar; otomatis membuat transaksi pengeluaran.",
			Parameters: schema(map[string]any{
				"name":   str("nama kebutuhan, mis. listrik"),
				"amount": integer("nominal kalau berbeda dari perkiraan (opsional)"),
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
			Description: "Ringkasan sebelum menutup bulan: total, budget, kebutuhan belum dibayar, hutang.",
			Parameters: schema(map[string]any{
				"month": str("bulan YYYY-MM, kosongkan untuk bulan ini"),
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
			Description: "Tutup bulan: kunci transaksi, simpan snapshot, siapkan bulan berikutnya. Aksi berisiko, selalu butuh konfirmasi pengguna.",
			Parameters: schema(map[string]any{
				"month": str("bulan YYYY-MM yang mau ditutup, kosongkan untuk bulan ini"),
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
			Description: "Buka kembali bulan yang sudah ditutup supaya bisa diedit lagi. Aksi berisiko, butuh konfirmasi pengguna.",
			Parameters: schema(map[string]any{
				"month": str("bulan YYYY-MM yang mau dibuka"),
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

// runTool menjalankan satu tool; tool berisiko hanya lewat alur konfirmasi.
func (a *Agent) runTool(ctx context.Context, name string, args json.RawMessage, confirmed bool) (string, error) {
	def, found := a.toolByName(name)
	if !found {
		return "", fmt.Errorf("tool %s tidak dikenal", name)
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

// amountOf menerima angka dari model bisa berupa number atau string.
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
