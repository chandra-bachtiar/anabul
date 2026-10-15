package agent

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"anabul/internal/tools"
)

// parsedIntent is what the simple parser understood from a message. Kept apart
// from execution so it can be tested without a database.
type parsedIntent struct {
	Tool      string `json:"tool"`
	Amount    int64  `json:"amount,omitempty"`
	Note      string `json:"note,omitempty"`
	Type      string `json:"type,omitempty"`
	Person    string `json:"person,omitempty"`
	Direction string `json:"direction,omitempty"`
	Category  string `json:"category,omitempty"`
	Month     string `json:"month,omitempty"`
	// Date holds "YYYY-MM-DD" when the message names another day, e.g.
	// "kopi kemarin 25rb".
	Date string `json:"date,omitempty"`
}

var (
	amountRe   = regexp.MustCompile(`(\d[\d.,]*\s*(?:rb|ribu|jt|juta|k\b)?)`)
	trailingRe = regexp.MustCompile(`\s+`)

	// The whole "sudah dibayar" phrase has to match, otherwise stray letters
	// survive in the need name.
	paidPhraseRe = regexp.MustCompile(`\b(?:sudah|udah|telah)?\s*(?:dibayar(?:in|kan)?|bayar|lunas|terbayar)\b`)
	fillerWords  = []string{"dong", "nih", "ya", "sih", "deh", "kah"}

	yesterdayRe  = regexp.MustCompile(`\b(kemarin|kmrn|semalam)\b`)
	todayRe      = regexp.MustCompile(`\b(hari ini|tadi|barusan|baru aja)\b`)
	dayOfMonthRe = regexp.MustCompile(`\b(?:tanggal|tgl)\s+(\d{1,2})\b`)
)

var incomeWords = []string{"gajian", "gaji", "bonus", "thr", "pemasukan", "dapat", "terima", "masuk"}
var debtWords = []string{"minjem", "minjam", "pinjam", "ngebon", "utang", "hutang", "piutang"}
var paidWords = []string{"udah dibayar", "sudah dibayar", "udah bayar", "sudah bayar", "lunas", "terbayar", "udah dibayarin"}
var paymentWords = []string{"bayar", "nyicil", "cicil", "melunasi"}

// parseIntent reads the message with plain rules. It runs when the LLM is not
// configured and stays as the safety net when the model is unreachable.
func parseIntent(message string, now time.Time) parsedIntent {
	raw := strings.TrimSpace(message)
	lower := strings.ToLower(raw)
	date := dateFromMessage(lower, now)
	// Time words go first so the digits in "tgl 5" are not read as an amount.
	withoutTime := stripTimeWords(lower)

	// "Budi udah bayar 100rb" / "bayar hutang Budi 100rb"
	if containsAny(lower, paymentWords) && !isNeedPayment(lower) {
		if person, ok := personBeforePayment(lower); ok {
			if amount, ok := lastAmount(withoutTime); ok {
				return parsedIntent{Tool: "add_debt_payment", Person: person, Amount: amount, Date: date}
			}
		}
	}

	// "listrik udah dibayar"
	if containsAny(lower, paidWords) {
		if name := needName(lower); name != "" {
			amount := int64(0)
			if parsed, ok := lastAmount(withoutTime); ok {
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
	if amount, ok := lastAmount(withoutTime); ok {
		intent := parsedIntent{
			Tool:   "add_transaction",
			Amount: amount,
			Type:   "expense",
			Note:   noteOf(raw, amount),
			Date:   date,
		}
		if containsAny(lower, incomeWords) {
			intent.Type = "income"
		}
		return intent
	}

	return parsedIntent{}
}

// dateFromMessage reads simple time words: "kemarin", "hari ini", "tadi", or
// "tanggal 12", and returns "YYYY-MM-DD" in the application timezone.
//
// For "tanggal N" the current month wins while that day has not passed yet;
// once it has, the previous month wins, because people usually write the entry
// after the fact.
func dateFromMessage(lower string, now time.Time) string {
	switch {
	case yesterdayRe.MatchString(lower):
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case dayOfMonthRe.MatchString(lower):
		matched := dayOfMonthRe.FindStringSubmatch(lower)
		day, err := strconv.Atoi(matched[1])
		if err != nil || day < 1 || day > 31 {
			return ""
		}
		thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		if day <= now.Day() {
			return tools.DueDateFromDay(thisMonth, day).Format("2006-01-02")
		}
		return tools.DueDateFromDay(thisMonth.AddDate(0, -1, 0), day).Format("2006-01-02")
	case todayRe.MatchString(lower):
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
			OccurredAt: timeFromDate(intent.Date, now),
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
			Person:            intent.Person,
			Amount:            intent.Amount,
			PaidAt:            timeFromDate(intent.Date, now),
			RecordTransaction: true,
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
		biggest := summary.BiggestDrain[0]
		reply += "\nPaling banyak: " + biggest.CategoryName + " " + tools.FormatAmount(biggest.Spent)
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

// lastAmount takes the last amount in the message: "kopi 25rb" -> 25000.
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

// noteOf drops the amount and any time words from the note: "kopi kemarin 25rb"
// -> "kopi".
func noteOf(original string, amount int64) string {
	lower := strings.ToLower(original)
	matches := amountRe.FindAllStringIndex(lower, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lower[matches[i][0]:matches[i][1]])
		if parsed, err := tools.ParseAmount(candidate); err == nil && parsed == amount {
			note := strings.TrimSpace(original[:matches[i][0]] + original[matches[i][1]:])
			return stripTimeWords(note)
		}
	}
	return stripTimeWords(strings.TrimSpace(original))
}

func stripTimeWords(note string) string {
	cleaned := yesterdayRe.ReplaceAllString(note, " ")
	cleaned = todayRe.ReplaceAllString(cleaned, " ")
	cleaned = dayOfMonthRe.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(trailingRe.ReplaceAllString(cleaned, " "))
}

// timeFromDate keeps the current clock time and swaps in the parsed date, so an
// entry written today about yesterday still lands at the right moment. Returns
// nil for an empty date, which makes the tool use the current time.
func timeFromDate(date string, now time.Time) *time.Time {
	date = strings.TrimSpace(date)
	if date == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", date, now.Location())
	if err != nil {
		return nil
	}
	combined := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), 0, now.Location())
	return &combined
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

// needName reads the need out of a message like "listrik udah dibayar".
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

// borrowingIntent separates "gua minjem 200rb ke Budi" (a debt) from
// "Budi minjem 150rb" (a receivable).
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
