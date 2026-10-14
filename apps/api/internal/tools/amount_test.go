package tools

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int64
		err   bool
	}{
		{name: "angka polos", input: "25000", want: 25000},
		{name: "titik ribuan", input: "25.000", want: 25000},
		{name: "juta penuh", input: "1.500.000", want: 1500000},
		{name: "sufiks rb", input: "25rb", want: 25000},
		{name: "sufiks rb spasi", input: "25 rb", want: 25000},
		{name: "sufiks ribu", input: "12 ribu", want: 12000},
		{name: "sufiks jt", input: "5jt", want: 5000000},
		{name: "sufiks juta", input: "5 juta", want: 5000000},
		{name: "desimal koma", input: "1,5jt", want: 1500000},
		{name: "desimal titik", input: "2.5jt", want: 2500000},
		{name: "prefix rp", input: "Rp 12.500", want: 12500},
		{name: "sufiks k", input: "35k", want: 35000},
		{name: "nol ditolak", input: "0", err: true},
		{name: "kosong ditolak", input: "", err: true},
		{name: "bukan angka", input: "kopi", err: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAmount(tc.input)
			if tc.err {
				if err == nil {
					t.Fatalf("ParseAmount(%q) ingin error, dapat %d", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAmount(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseAmount(%q) = %d, ingin %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseMonth(t *testing.T) {
	loc := time.FixedZone("WIB", 7*60*60)
	now := time.Date(2026, time.October, 13, 9, 0, 0, 0, loc)

	cases := []struct {
		name  string
		input string
		want  string
		err   bool
	}{
		{name: "bulan berjalan", input: "", want: "2026-10-01"},
		{name: "bulan eksplisit", input: "2026-07", want: "2026-07-01"},
		{name: "bulan desember", input: "2025-12", want: "2025-12-01"},
		{name: "bulan tidak valid", input: "2026-13", err: true},
		{name: "format salah", input: "oktober", err: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMonth(tc.input, now, loc)
			if tc.err {
				if err == nil {
					t.Fatalf("ParseMonth(%q) ingin error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMonth(%q) error: %v", tc.input, err)
			}
			if got.Format("2006-01-02") != tc.want {
				t.Fatalf("ParseMonth(%q) = %s, ingin %s", tc.input, got.Format("2006-01-02"), tc.want)
			}
		})
	}
}

func TestBudgetStatusAndRemaining(t *testing.T) {
	cases := []struct {
		name    string
		amount  int64
		spent   int64
		status  string
		remain  int64
		percent int
	}{
		{name: "aman", amount: 100000, spent: 45000, status: StatusOK, remain: 55000, percent: 45},
		{name: "hampir habis di 80 persen", amount: 100000, spent: 80000, status: StatusWarn, remain: 20000, percent: 80},
		{name: "lewat sedikit", amount: 100000, spent: 100001, status: StatusOver, remain: -1, percent: 100},
		{name: "lewat banyak", amount: 200000, spent: 350000, status: StatusOver, remain: -150000, percent: 175},
		{name: "tanpa budget", amount: 0, spent: 25000, status: StatusNone, remain: -25000, percent: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BudgetStatus(tc.amount, tc.spent); got != tc.status {
				t.Fatalf("BudgetStatus = %s, ingin %s", got, tc.status)
			}
			if got := BudgetRemaining(tc.amount, tc.spent); got != tc.remain {
				t.Fatalf("BudgetRemaining = %d, ingin %d", got, tc.remain)
			}
			if got := progress(tc.amount, tc.spent); got != tc.percent {
				t.Fatalf("progress = %d, ingin %d", got, tc.percent)
			}
		})
	}
}

func TestMood(t *testing.T) {
	cases := []struct {
		name    string
		locked  bool
		budget  int64
		spent   int64
		anyOver bool
		want    string
	}{
		{name: "bulan terkunci tidur", locked: true, budget: 100000, spent: 10000, want: moodTidur},
		{name: "pemakaian rendah senang", budget: 100000, spent: 30000, want: moodSenang},
		{name: "tepat 80 persen khawatir", budget: 100000, spent: 80000, want: moodKhawatir},
		{name: "kategori lewat khawatir", budget: 500000, spent: 100000, anyOver: true, want: moodKhawatir},
		{name: "tanpa budget senang", budget: 0, spent: 250000, want: moodSenang},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Mood(tc.locked, tc.budget, tc.spent, tc.anyOver); got != tc.want {
				t.Fatalf("Mood = %s, ingin %s", got, tc.want)
			}
		})
	}
}

func TestMatchCategory(t *testing.T) {
	categories := []Category{
		{ID: 1, Name: "makan-minum", Keywords: []string{"makan", "kopi", "jajan", "warung"}},
		{ID: 2, Name: "transport", Keywords: []string{"bensin", "ojek", "parkir"}},
		{ID: 3, Name: "tagihan", Keywords: []string{"listrik", "internet", "wifi"}},
	}

	cases := []struct {
		name string
		note string
		want int64
	}{
		{name: "keyword kopi", note: "kopi susu", want: 1},
		{name: "keyword bensin", note: "isi bensin motor", want: 2},
		{name: "keyword listrik", note: "bayar listrik", want: 3},
		{name: "nama kategori langsung", note: "transport harian", want: 2},
		{name: "tanpa kecocokan", note: "sesuatu yang aneh", want: 0},
		{name: "catatan kosong", note: "   ", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			match := MatchCategory(tc.note, categories)
			var got int64
			if match != nil {
				got = match.ID
			}
			if got != tc.want {
				t.Fatalf("MatchCategory(%q) = %d, ingin %d", tc.note, got, tc.want)
			}
		})
	}
}

func TestIsLocked(t *testing.T) {
	closed := time.Date(2026, time.November, 1, 9, 0, 0, 0, time.UTC)
	reopenedLater := closed.Add(2 * time.Hour)
	reopenedEarlier := closed.Add(-time.Hour)

	cases := []struct {
		name     string
		closedAt pgtype.Timestamptz
		reopened pgtype.Timestamptz
		want     bool
	}{
		{name: "belum pernah ditutup", want: false},
		{
			name:     "ditutup dan belum dibuka",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			want:     true,
		},
		{
			name:     "sudah dibuka kembali",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			reopened: pgtype.Timestamptz{Time: reopenedLater, Valid: true},
			want:     false,
		},
		{
			name:     "ditutup ulang setelah dibuka",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			reopened: pgtype.Timestamptz{Time: reopenedEarlier, Valid: true},
			want:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsLocked(tc.closedAt, tc.reopened); got != tc.want {
				t.Fatalf("IsLocked = %v, ingin %v", got, tc.want)
			}
		})
	}
}

func TestDueDateFromDay(t *testing.T) {
	loc := time.FixedZone("WIB", 7*60*60)

	cases := []struct {
		name  string
		month time.Time
		day   int
		want  string
	}{
		{name: "hari biasa", month: time.Date(2026, time.October, 1, 0, 0, 0, 0, loc), day: 5, want: "2026-10-05"},
		{name: "februari dipangkas", month: time.Date(2026, time.February, 1, 0, 0, 0, 0, loc), day: 31, want: "2026-02-28"},
		{name: "april 30 hari", month: time.Date(2026, time.April, 1, 0, 0, 0, 0, loc), day: 31, want: "2026-04-30"},
		{name: "kabisat", month: time.Date(2028, time.February, 1, 0, 0, 0, 0, loc), day: 30, want: "2028-02-29"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DueDateFromDay(tc.month, tc.day)
			if got.Format("2006-01-02") != tc.want {
				t.Fatalf("DueDateFromDay = %s, ingin %s", got.Format("2006-01-02"), tc.want)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		amount int64
		want   string
	}{
		{amount: 0, want: "Rp 0"},
		{amount: 900, want: "Rp 900"},
		{amount: 25000, want: "Rp 25.000"},
		{amount: 1500000, want: "Rp 1.500.000"},
		{amount: -15000, want: "-Rp 15.000"},
	}

	for _, tc := range cases {
		if got := FormatAmount(tc.amount); got != tc.want {
			t.Fatalf("FormatAmount(%d) = %s, ingin %s", tc.amount, got, tc.want)
		}
	}
}
