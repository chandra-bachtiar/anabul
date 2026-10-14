package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"anabul/internal/tools"
)

// systemPrompt menyusun konteks yang dikirim ke model tiap request.
func (a *Agent) systemPrompt(ctx context.Context, channel string) string {
	var b strings.Builder
	now := a.tools.Now()

	b.WriteString("Kamu ANABUL, asisten kucing yang ramah untuk mencatat keuangan pribadi pemilikmu.\n")
	b.WriteString("Balas singkat dalam bahasa Indonesia santai, maksimal satu emoji, boleh sesekali bilang \"meong\".\n")
	b.WriteString("Angka dan status selalu ditulis jelas. Untuk error, konfirmasi aksi berisiko, dan tutup bulan: hilangkan candaan.\n")
	b.WriteString("Kalau ada informasi penting yang kurang (misal nominal), tanya balik satu pertanyaan pendek. Jangan menebak.\n\n")

	fmt.Fprintf(&b, "Tanggal hari ini: %s (%s), bulan berjalan %s.\n", now.Format("2006-01-02"), a.tools.Location().String(), tools.FormatMonth(a.tools.CurrentMonth()))
	fmt.Fprintf(&b, "Kanal percakapan: %s.\n", channel)

	if categories, err := a.tools.ListCategories(ctx); err == nil && len(categories) > 0 {
		names := make([]string, 0, len(categories))
		for _, category := range categories {
			names = append(names, category.Name)
		}
		fmt.Fprintf(&b, "Kategori tersedia: %s.\n", strings.Join(names, ", "))
	}
	if accounts, err := a.tools.ListAccounts(ctx); err == nil && len(accounts) > 0 {
		names := make([]string, 0, len(accounts))
		for _, account := range accounts {
			names = append(names, account.Name)
		}
		fmt.Fprintf(&b, "Akun tersedia: %s.\n", strings.Join(names, ", "))
	}

	month := a.tools.CurrentMonth()
	if isLocked, err := a.tools.IsLocked(ctx, month); err == nil {
		if isLocked {
			fmt.Fprintf(&b, "Status bulan %s: TERKUNCI. Semua perubahan data bulan ini ditolak, tawarkan buka kembali lewat konfirmasi.\n", tools.FormatMonth(month))
		} else {
			fmt.Fprintf(&b, "Status bulan %s: terbuka.\n", tools.FormatMonth(month))
		}
	}

	if needs, err := a.tools.ListMonthlyNeeds(ctx, tools.FormatMonth(month), "unpaid"); err == nil && len(needs) > 0 {
		items := make([]string, 0, len(needs))
		for _, need := range needs {
			items = append(items, fmt.Sprintf("%s %s (%s)", need.Name, tools.FormatAmount(need.Amount), dueLabel(need)))
		}
		fmt.Fprintf(&b, "Kebutuhan belum dibayar bulan ini: %s.\n", strings.Join(items, "; "))
	}

	if debts, err := a.tools.ListDebts(ctx, "open", ""); err == nil && len(debts) > 0 {
		items := make([]string, 0, len(debts))
		for _, debt := range debts {
			items = append(items, fmt.Sprintf("%s %s %s (id %d)", debt.Direction, debt.Person, tools.FormatAmount(debt.Remaining), debt.ID))
		}
		fmt.Fprintf(&b, "Hutang/piutang berjalan: %s.\n", strings.Join(items, "; "))
	}

	if summary, err := a.tools.MonthSummary(ctx, ""); err == nil {
		fmt.Fprintf(&b, "Ringkas bulan ini: pemasukan %s, pengeluaran %s, sisa budget %s.\n",
			tools.FormatAmount(summary.Income), tools.FormatAmount(summary.Expense), tools.FormatAmount(summary.BudgetLeft))
	}

	b.WriteString("\nAturan: panggil tool untuk setiap perubahan data. Aksi berisiko (hapus transaksi, tutup bulan, buka bulan) tidak dieksekusi langsung; tool akan mengembalikan needs_confirmation dan pengguna yang memutuskan.\n")
	b.WriteString("Nominal selalu integer rupiah tanpa titik atau koma.\n")
	b.WriteString("Kalau pengguna menyebut waktu lain (mis. \"kemarin\", \"tgl 5\"), isi argumen date dengan format YYYY-MM-DD sesuai tanggal hari ini di atas. Kalau tidak disebut, biarkan kosong supaya dicatat hari ini.\n")
	return b.String()
}

func dueLabel(need tools.NeedRow) string {
	switch {
	case need.Overdue:
		return "telat " + need.DueDate
	case need.DueSoon:
		return "segera " + need.DueDate
	default:
		return "jatuh tempo " + need.DueDate
	}
}

// historyMessages mengambil beberapa pesan terakhir untuk konteks koreksi.
func (a *Agent) historyMessages(ctx context.Context, channel string, limit int32) []chatMessage {
	rows, err := a.q.ListRecentChatMessages(ctx, storeListRecentParams(channel, limit))
	if err != nil {
		return nil
	}
	result := make([]chatMessage, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		result = append(result, chatMessage{Role: row.Role, Content: row.Content})
	}
	return result
}

func trimHistory(messages []chatMessage, max int) []chatMessage {
	if len(messages) <= max {
		return messages
	}
	return messages[len(messages)-max:]
}

func timeNow(loc *time.Location) time.Time {
	return time.Now().In(loc)
}
