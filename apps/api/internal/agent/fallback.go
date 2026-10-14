package agent

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"anabul/internal/tools"
)

// parsedIntent adalah hasil pembacaan pesan oleh parser sederhana. Dipisahkan
// dari eksekusi supaya bisa diuji tanpa database.
type parsedIntent struct {
	Tool      string `json:"tool"`
	Amount    int64  `json:"amount,omitempty"`
	Note      string `json:"note,omitempty"`
	Type      string `json:"type,omitempty"`
	Person    string `json:"person,omitempty"`
	Direction string `json:"direction,omitempty"`
	Category  string `json:"category,omitempty"`
	Month     string `json:"month,omitempty"`
	// Date berisi tanggal "YYYY-MM-DD" kalau pesan menyebut waktu lain,
	// mis. "kopi kemarin 25rb".
	Date string `json:"date,omitempty"`
}

var (
	amountRe   = regexp.MustCompile(`(\d[\d.,]*\s*(?:rb|ribu|jt|juta|k\b)?)`)
	trailingRe = regexp.MustCompile(`\s+`)

	// Frasa "sudah dibayar" harus dibaca utuh supaya tidak menyisakan huruf.
	paidPhraseRe = regexp.MustCompile(`\b(?:sudah|udah|telah)?\s*(?:dibayar(?:in|kan)?|bayar|lunas|terbayar)\b`)
	fillerWords  = []string{"dong", "nih", "ya", "sih", "deh", "kah"}

	waktuKemarinRe = regexp.MustCompile(`\b(kemarin|kmrn|semalam)\b`)
	waktuHariIniRe = regexp.MustCompile(`\b(hari ini|tadi|barusan|baru aja)\b`)
	waktuTanggalRe = regexp.MustCompile(`\b(?:tanggal|tgl)\s+(\d{1,2})\b`)
)

var incomeWords = []string{"gajian", "gaji", "bonus", "thr", "pemasukan", "dapat", "terima", "masuk"}
var debtWords = []string{"minjem", "minjam", "pinjam", "ngebon", "utang", "hutang", "piutang"}
var paidWords = []string{"udah dibayar", "sudah dibayar", "udah bayar", "sudah bayar", "lunas", "terbayar", "udah dibayarin"}
var paymentWords = []string{"bayar", "nyicil", "cicil", "melunasi"}

// parseIntent membaca maksud pesan dengan aturan sederhana. Dipakai kalau LLM
// belum dikonfigurasi, dan jadi jaring pengaman kalau model tidak tersedia.
// Waktu "sekarang" dikirim dari pemanggil supaya hasilnya bisa diuji tetap.
func parseIntent(message string, now time.Time) parsedIntent {
	raw := strings.TrimSpace(message)
	lower := strings.ToLower(raw)
	tanggal := dateFromMessage(lower, now)
	// Kata waktu dibuang dulu supaya angka di "tgl 5" tidak dikira nominal.
	tanpaWaktu := buangKeteranganWaktu(lower)

	// "Budi udah bayar 100rb" / "bayar hutang Budi 100rb"
	if containsAny(lower, paymentWords) && !isNeedPayment(lower) {
		if person, ok := personBeforePayment(lower); ok {
			if amount, ok := lastAmount(tanpaWaktu); ok {
				return parsedIntent{Tool: "add_debt_payment", Person: person, Amount: amount, Date: tanggal}
			}
		}
	}

	// "listrik udah dibayar"
	if containsAny(lower, paidWords) {
		if name := needName(lower); name != "" {
			amount := int64(0)
			if parsed, ok := lastAmount(tanpaWaktu); ok {
				amount = parsed
			}
			intent := parsedIntent{Tool: "mark_need_paid", Note: name}
			if amount > 0 {
				intent.Amount = amount
			}
			return intent
		}
	}

	// "gua minjem 200rb ke Budi" / "Budi minjem 150rb"
	if containsAny(lower, debtWords) {
		if intent, ok := borrowingIntent(lower); ok {
			return intent
		}
	}

	// "sisa budget makan berapa?"
	if strings.Contains(lower, "budget") && containsAny(lower, []string{"sisa", "berapa", "tinggal"}) {
		return parsedIntent{Tool: "get_budget_remaining", Category: budgetCategory(lower)}
	}

	// "bulan ini boros di mana?"
	if containsAny(lower, []string{"boros", "ringkasan", "summary", "rekap", "habis di mana"}) {
		return parsedIntent{Tool: "get_summary"}
	}

	// "tutup bulan"
	if strings.Contains(lower, "tutup bulan") {
		return parsedIntent{Tool: "close_month"}
	}

	// "kopi 25rb", "makan siang 35000", "gajian 5jt", "kopi kemarin 25rb"
	if amount, ok := lastAmount(tanpaWaktu); ok {
		intent := parsedIntent{
			Tool:   "add_transaction",
			Amount: amount,
			Type:   "expense",
			Note:   noteOf(raw, amount),
			Date:   tanggal,
		}
		if containsAny(lower, incomeWords) {
			intent.Type = "income"
		}
		return intent
	}

	return parsedIntent{}
}

// dateFromMessage membaca keterangan waktu sederhana: "kemarin", "hari ini",
// "tadi", atau "tanggal 12". Mengembalikan tanggal "YYYY-MM-DD" di zona lokasi
// yang dipakai aplikasi, atau string kosong kalau tidak ada keterangan waktu.
//
// Aturan "tanggal N": kalau harinya belum lewat di bulan ini, yang dimaksud
// adalah tanggal N bulan ini; kalau sudah lewat, dipakai bulan sebelumnya karena
// catatan biasanya ditulis setelah kejadiannya.
func dateFromMessage(lower string, now time.Time) string {
	switch {
	case waktuKemarinRe.MatchString(lower):
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case waktuTanggalRe.MatchString(lower):
		cocok := waktuTanggalRe.FindStringSubmatch(lower)
		hari, err := strconv.Atoi(cocok[1])
		if err != nil || hari < 1 || hari > 31 {
			return ""
		}
		bulanIni := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		if hari <= now.Day() {
			return tools.DueDateFromDay(bulanIni, hari).Format("2006-01-02")
		}
		return tools.DueDateFromDay(bulanIni.AddDate(0, -1, 0), hari).Format("2006-01-02")
	case waktuHariIniRe.MatchString(lower):
		return now.Format("2006-01-02")
	default:
		return ""
	}
}

func (a *Agent) runParser(ctx context.Context, channel, message string) (string, []map[string]any, error) {
	now := a.tools.Now()
	intent := parseIntent(message, now)
	log := make([]map[string]any, 0, 1)

	switch intent.Tool {
	case "add_transaction":
		tx, err := a.tools.AddTransaction(ctx, tools.AddTransactionInput{
			Type:       intent.Type,
			Amount:     intent.Amount,
			Note:       intent.Note,
			OccurredAt: waktuDariTanggal(intent.Date, now),
			Source:     normalizeChannel(channel),
		})
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("add_transaction", intent, tx))
		label := "Pengeluaran"
		arrow := "-"
		if tx.Type == "income" {
			label = "Pemasukan"
			arrow = "+"
		}
		reply := label + " " + tools.FormatAmount(tx.Amount) + " tercatat (" + arrow + ")."
		if tx.Note != "" {
			reply = label + " " + tools.FormatAmount(tx.Amount) + " buat " + tx.Note + " tercatat."
		}
		if tx.CategoryName != "" {
			reply += " Kategori: " + tx.CategoryName + "."
		}
		return reply + " Meong!", log, nil
	case "add_debt":
		debt, err := a.tools.AddDebt(ctx, tools.AddDebtInput{
			Person:    intent.Person,
			Direction: intent.Direction,
			Amount:    intent.Amount,
			Note:      intent.Note,
		})
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("add_debt", intent, debt))
		if debt.Direction == "piutang" {
			return intent.Person + " berhutang " + tools.FormatAmount(debt.AmountTotal) + " ke kamu, sudah dicatat.", log, nil
		}
		return "Hutang ke " + intent.Person + " " + tools.FormatAmount(debt.AmountTotal) + " sudah dicatat.", log, nil
	case "add_debt_payment":
		debt, err := a.tools.AddDebtPayment(ctx, tools.AddDebtPaymentInput{
			Person:         intent.Person,
			Amount:         intent.Amount,
			PaidAt:         waktuDariTanggal(intent.Date, now),
			CatatTransaksi: true,
		})
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("add_debt_payment", intent, debt))
		if debt.Status == "settled" {
			return "Pembayaran " + tools.FormatAmount(intent.Amount) + " dari " + debt.Person + " dicatat. Sisa sudah lunas.", log, nil
		}
		return "Pembayaran " + tools.FormatAmount(intent.Amount) + " dari " + debt.Person + " dicatat. Sisa " + tools.FormatAmount(debt.Remaining) + ".", log, nil
	case "mark_need_paid":
		need, err := a.tools.FindNeedByName(ctx, a.tools.CurrentMonth(), intent.Note)
		if err != nil {
			return friendlyError(err), log, nil
		}
		var amount *int64
		if intent.Amount > 0 {
			amount = &intent.Amount
		}
		paid, err := a.tools.MarkNeedPaid(ctx, need.ID, amount)
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("mark_need_paid", intent, paid))
		return paid.Name + " sudah ditandai lunas (" + tools.FormatAmount(paid.Amount) + ") dan masuk sebagai pengeluaran.", log, nil
	case "get_budget_remaining":
		rows, err := a.tools.ListBudgets(ctx, "")
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("get_budget_remaining", intent, rows))
		return budgetReply(rows, intent.Category), log, nil
	case "get_summary":
		summary, err := a.tools.MonthSummary(ctx, "")
		if err != nil {
			return friendlyError(err), log, nil
		}
		log = append(log, toolLogEntry("get_summary", intent, summary))
		return summaryReply(summary), log, nil
	case "close_month":
		return "Tutup bulan lewat halaman Lainnya ya, di situ ada pratinjau dan tombol konfirmasinya.", log, nil
	default:
		return "Meong, nominalnya berapa? Contoh: \"kopi 25rb\" atau \"gua pinjam 200rb ke Budi\".", log, nil
	}
}

func budgetReply(rows []tools.BudgetRow, category string) string {
	if len(rows) == 0 {
		return "Belum ada budget yang diatur bulan ini. Buka tab Budget buat mengisinya ya."
	}
	var picked []tools.BudgetRow
	for _, row := range rows {
		if category == "" || strings.Contains(strings.ToLower(row.CategoryName), category) {
			picked = append(picked, row)
		}
	}
	if len(picked) == 0 {
		return "Kategori " + category + " belum punya budget bulan ini."
	}
	lines := make([]string, 0, len(picked))
	for _, row := range picked {
		status := "aman"
		switch row.Status {
		case tools.StatusWarn:
			status = "hampir habis"
		case tools.StatusOver:
			status = "lewat"
		case tools.StatusNone:
			status = "tanpa budget"
		}
		lines = append(lines, row.CategoryName+": sisa "+tools.FormatAmount(row.Remaining)+" ("+status+")")
	}
	return "Sisa budget bulan ini\n" + strings.Join(lines, "\n")
}

func summaryReply(summary tools.Summary) string {
	reply := "Ringkasan " + summary.MonthLabel + "\n"
	reply += "Pemasukan " + tools.FormatAmount(summary.Income) + "\n"
	reply += "Pengeluaran " + tools.FormatAmount(summary.Expense) + "\n"
	reply += "Selisih " + tools.FormatAmount(summary.Net)
	if len(summary.BiggestDrain) > 0 {
		paling := summary.BiggestDrain[0]
		reply += "\nPaling banyak: " + paling.CategoryName + " " + tools.FormatAmount(paling.Spent)
	}
	return reply
}

func toolLogEntry(name string, intent parsedIntent, result any) map[string]any {
	return map[string]any{"name": name, "args": intent, "result": result}
}

func containsAny(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// lastAmount mengambil nominal terakhir di pesan, mis. "kopi 25rb" -> 25000.
func lastAmount(text string) (int64, bool) {
	matches := amountRe.FindAllString(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(matches[i])
		value, err := tools.ParseAmount(candidate)
		if err == nil {
			return value, true
		}
	}
	return 0, false
}

// noteOf mengambil catatan dengan membuang nominal dan keterangan waktu dari
// pesan, mis. "kopi kemarin 25rb" -> "kopi".
func noteOf(original string, amount int64) string {
	lower := strings.ToLower(original)
	matches := amountRe.FindAllStringIndex(lower, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lower[matches[i][0]:matches[i][1]])
		if parsed, err := tools.ParseAmount(candidate); err == nil && parsed == amount {
			note := strings.TrimSpace(original[:matches[i][0]] + original[matches[i][1]:])
			return buangKeteranganWaktu(note)
		}
	}
	return buangKeteranganWaktu(strings.TrimSpace(original))
}

// buangKeteranganWaktu menghapus kata waktu dari catatan supaya tidak ikut
// tersimpan sebagai nama pengeluaran.
func buangKeteranganWaktu(note string) string {
	bersih := waktuKemarinRe.ReplaceAllString(note, " ")
	bersih = waktuHariIniRe.ReplaceAllString(bersih, " ")
	bersih = waktuTanggalRe.ReplaceAllString(bersih, " ")
	return strings.TrimSpace(trailingRe.ReplaceAllString(bersih, " "))
}

// waktuDariTanggal menggabungkan tanggal "YYYY-MM-DD" dengan jam saat ini di
// zona aplikasi. Mengembalikan nil kalau tanggalnya kosong atau tidak valid,
// supaya tool memakai waktu sekarang.
func waktuDariTanggal(tanggal string, now time.Time) *time.Time {
	tanggal = strings.TrimSpace(tanggal)
	if tanggal == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", tanggal, now.Location())
	if err != nil {
		return nil
	}
	gabung := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), 0, now.Location())
	return &gabung
}

func personBeforePayment(lower string) (string, bool) {
	for _, word := range paymentWords {
		index := strings.Index(lower, word)
		if index <= 0 {
			continue
		}
		candidate := strings.TrimSpace(lower[:index])
		candidate = strings.TrimPrefix(candidate, "ke ")
		candidate = strings.TrimSpace(strings.TrimSuffix(candidate, "udah"))
		candidate = strings.TrimSpace(strings.TrimSuffix(candidate, "sudah"))
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || containsAny(candidate, []string{"hutang", "utang", "piutang", "sisa", "budget"}) {
			continue
		}
		fields := strings.Fields(candidate)
		if len(fields) == 0 || len(fields) > 3 {
			continue
		}
		return titleCase(fields[len(fields)-1]), true
	}
	return "", false
}

func isNeedPayment(lower string) bool {
	return containsAny(lower, paidWords) && containsAny(lower, []string{"listrik", "air", "internet", "wifi", "pulsa", "kuota", "sewa", "cicilan"})
}

// needName mengambil nama kebutuhan dari pesan seperti "listrik udah dibayar".
func needName(lower string) string {
	cleaned := paidPhraseRe.ReplaceAllString(lower, " ")
	cleaned = amountRe.ReplaceAllString(cleaned, " ")
	for _, word := range fillerWords {
		cleaned = replaceWord(cleaned, word)
	}
	cleaned = strings.TrimSpace(trailingRe.ReplaceAllString(cleaned, " "))
	if cleaned == "" {
		return ""
	}
	return cleaned
}

func replaceWord(text, word string) string {
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return pattern.ReplaceAllString(text, " ")
}

// borrowingIntent memisahkan "gua minjem 200rb ke Budi" (hutang) dari
// "Budi minjem 150rb" (piutang).
func borrowingIntent(lower string) (parsedIntent, bool) {
	amount, ok := lastAmount(lower)
	if !ok {
		return parsedIntent{}, false
	}

	if index := strings.Index(lower, " ke "); index >= 0 {
		person := firstWord(lower[index+4:])
		if person != "" {
			return parsedIntent{Tool: "add_debt", Person: person, Direction: "hutang", Amount: amount}, true
		}
	}

	trimmed := strings.TrimSpace(amountRe.ReplaceAllString(lower, " "))
	for _, word := range debtWords {
		index := strings.Index(trimmed, word)
		if index < 0 {
			continue
		}
		before := strings.TrimSpace(trimmed[:index])
		before = strings.TrimSpace(strings.TrimPrefix(before, "gua "))
		before = strings.TrimSpace(strings.TrimPrefix(before, "aku "))
		before = strings.TrimSpace(strings.TrimPrefix(before, "saya "))
		before = strings.TrimSpace(strings.TrimPrefix(before, "aku lagi "))
		if before == "" {
			return parsedIntent{Tool: "add_debt", Person: "tanpa nama", Direction: "hutang", Amount: amount}, true
		}
		return parsedIntent{Tool: "add_debt", Person: titleCase(firstWord(before)), Direction: "piutang", Amount: amount}, true
	}
	return parsedIntent{}, false
}

func budgetCategory(lower string) string {
	cleaned := lower
	for _, word := range []string{"sisa", "budget", "berapa", "tinggal", "?", "dong", "ya", "nih", "bulan", "ini"} {
		cleaned = strings.ReplaceAll(cleaned, word, " ")
	}
	return strings.TrimSpace(trailingRe.ReplaceAllString(cleaned, " "))
}

func firstWord(text string) string {
	fields := strings.FieldsFunc(strings.TrimSpace(text), func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '?' || r == '!'
	})
	if len(fields) == 0 {
		return ""
	}
	return titleCase(fields[0])
}

func titleCase(word string) string {
	if word == "" {
		return ""
	}
	runes := []rune(word)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}
