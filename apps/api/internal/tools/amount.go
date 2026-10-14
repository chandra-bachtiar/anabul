package tools

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Status budget yang dipakai UI (warna) dan agent (kalimat singkat).
const (
	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusOver = "over"
	StatusNone = "none"
)

const (
	moodSenang   = "senang"
	moodKhawatir = "khawatir"
	moodTidur    = "tidur"
)

var amountPattern = regexp.MustCompile(`[0-9][0-9.,]*`)

// ParseAmount membaca nominal dari teks bebas: "25000", "25.000", "25rb",
// "5jt", "Rp 12 500", "1,5jt". Dipakai parser Telegram dan validasi tool.
func ParseAmount(raw string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return 0, invalid("nominalnya belum ada")
	}

	multiplier := int64(1)
	switch {
	case strings.Contains(s, "jt"), strings.Contains(s, "juta"), strings.Contains(s, "mio"):
		multiplier = 1_000_000
	case strings.Contains(s, "rb"), strings.Contains(s, "ribu"), strings.HasSuffix(s, "k"):
		multiplier = 1_000
	}

	digits := amountPattern.FindString(s)
	if digits == "" {
		return 0, invalid("nominal %q tidak bisa dibaca, coba tulis angkanya ya (mis. 25000)", raw)
	}

	value, err := parseDigits(digits, multiplier)
	if err != nil {
		return 0, err
	}

	total := int64(math.Round(value * float64(multiplier)))
	if total <= 0 {
		return 0, invalid("nominal harus lebih dari 0")
	}
	if total > 1_000_000_000_000 {
		return 0, invalid("nominalnya kegedean, cek lagi ya")
	}
	return total, nil
}

// parseDigits menangani pemisah ribuan Indonesia ("25.000", "1.500.000") dan
// desimal ("1,5" atau "1.5" saat ada satuan juta/ribu).
func parseDigits(digits string, multiplier int64) (float64, error) {
	hasComma := strings.Contains(digits, ",")
	hasDot := strings.Contains(digits, ".")

	normalized := digits
	switch {
	case hasComma:
		normalized = strings.ReplaceAll(digits, ".", "")
		normalized = strings.Replace(normalized, ",", ".", 1)
	case hasDot:
		parts := strings.Split(digits, ".")
		last := parts[len(parts)-1]
		decimal := len(parts) == 2 && len(last) <= 2 && multiplier > 1
		if decimal {
			break
		}
		normalized = strings.ReplaceAll(digits, ".", "")
	}

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, invalid("nominal %q tidak bisa dibaca", digits)
	}
	return value, nil
}

// ParseMonth membaca bulan "YYYY-MM"; kosong berarti bulan berjalan.
func ParseMonth(raw string, now time.Time, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return MonthStart(now.In(loc), loc), nil
	}
	parsed, err := time.ParseInLocation("2006-01", raw, loc)
	if err != nil {
		return time.Time{}, invalid("format bulan harus YYYY-MM, mis. 2026-10")
	}
	return parsed, nil
}

// MonthStart membulatkan waktu ke tanggal 1 bulan tersebut di zona lokasi.
func MonthStart(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
}

// MonthEnd adalah tanggal 1 bulan berikutnya (eksklusif).
func MonthEnd(month time.Time) time.Time {
	return month.AddDate(0, 1, 0)
}

// FormatMonth menulis bulan sebagai "YYYY-MM".
func FormatMonth(month time.Time) string {
	return month.Format("2006-01")
}

// FormatMonthLabel menulis bulan sebagai "Oktober 2026".
func FormatMonthLabel(month time.Time) string {
	return month.Format("January 2006")
}

// BudgetStatus menentukan status warna dari budget vs pemakaian.
func BudgetStatus(amount, spent int64) string {
	switch {
	case amount <= 0:
		return StatusNone
	case spent > amount:
		return StatusOver
	case spent*100 >= amount*80:
		return StatusWarn
	default:
		return StatusOK
	}
}

// BudgetRemaining menghitung sisa; boleh negatif kalau lewat.
func BudgetRemaining(amount, spent int64) int64 { return amount - spent }

// Mood maskot di Home: tidur saat bulan terkunci, khawatir saat budget
// menipis/lewat, selebihnya senang.
func Mood(locked bool, totalBudget, totalSpent int64, anyOver bool) string {
	if locked {
		return moodTidur
	}
	if anyOver || (totalBudget > 0 && totalSpent*100 >= totalBudget*80) {
		return moodKhawatir
	}
	return moodSenang
}

// MatchCategory mencari kategori paling cocok untuk sebuah catatan.
func MatchCategory(note string, candidates []Category) *Category {
	lowered := strings.ToLower(note)
	if strings.TrimSpace(lowered) == "" {
		return nil
	}
	words := strings.FieldsFunc(lowered, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}

	for i := range candidates {
		c := &candidates[i]
		if strings.Contains(lowered, strings.ToLower(c.Name)) {
			return c
		}
		for _, keyword := range c.Keywords {
			k := strings.ToLower(strings.TrimSpace(keyword))
			if k == "" {
				continue
			}
			if wordSet[k] {
				return c
			}
			if len(k) >= 4 && strings.Contains(lowered, k) {
				return c
			}
		}
	}
	return nil
}

func dateOf(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func dateOfPtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func timeOfPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func int8Ptr(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

func textPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func boolPtr(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func int16Ptr(v *int16) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *v, Valid: true}
}

func int8Value(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

func dateValue(v pgtype.Date) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

// timeValue mengubah pgtype.Timestamptz menjadi waktu di zona aplikasi.
func timeValue(v pgtype.Timestamptz, loc *time.Location) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.In(loc)
	return &t
}

func textValue(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func derefStrings(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

// pgDateToTime mengubah pgtype.Date menjadi waktu lokal zona aplikasi.
func pgDateToTime(v pgtype.Date, loc *time.Location) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return time.Date(v.Time.Year(), v.Time.Month(), v.Time.Day(), 0, 0, 0, 0, loc)
}

// StartOfDay membulatkan waktu ke pukul 00:00 di zona lokasi.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// DueDateFromDay menghitung tanggal jatuh tempo untuk hari ke-N, dibatasi ke
// hari terakhir bulan tersebut (mis. due_day 31 di Februari).
func DueDateFromDay(month time.Time, day int) time.Time {
	last := MonthEnd(month).AddDate(0, 0, -1)
	if day > last.Day() {
		day = last.Day()
	}
	if day < 1 {
		day = 1
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, month.Location())
}

// FormatAmount dipakai agent untuk menulis nominal: 25000 -> "Rp 25.000".
func FormatAmount(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	raw := strconv.FormatInt(amount, 10)
	parts := make([]string, 0, len(raw)/3+1)
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	formatted := "Rp " + strings.Join(parts, ".")
	if negative {
		return "-" + formatted
	}
	return formatted
}

// IsLocked menentukan status kunci dari baris month_closings.
func IsLocked(closedAt, reopenedAt pgtype.Timestamptz) bool {
	if !closedAt.Valid {
		return false
	}
	if !reopenedAt.Valid {
		return true
	}
	return !reopenedAt.Time.After(closedAt.Time)
}
