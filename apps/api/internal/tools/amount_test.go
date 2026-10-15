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
		{name: "plain digits", input: "25000", want: 25000},
		{name: "thousands separator", input: "25.000", want: 25000},
		{name: "full millions", input: "1.500.000", want: 1500000},
		{name: "rb suffix", input: "25rb", want: 25000},
		{name: "rb suffix with space", input: "25 rb", want: 25000},
		{name: "ribu suffix", input: "12 ribu", want: 12000},
		{name: "jt suffix", input: "5jt", want: 5000000},
		{name: "juta suffix", input: "5 juta", want: 5000000},
		{name: "comma decimal", input: "1,5jt", want: 1500000},
		{name: "dot decimal", input: "2.5jt", want: 2500000},
		{name: "rp prefix", input: "Rp 12.500", want: 12500},
		{name: "k suffix", input: "35k", want: 35000},
		{name: "zero rejected", input: "0", err: true},
		{name: "empty rejected", input: "", err: true},
		{name: "not a number", input: "kopi", err: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAmount(tc.input)
			if tc.err {
				if err == nil {
					t.Fatalf("ParseAmount(%q) wanted an error, got %d", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAmount(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseAmount(%q) = %d, want %d", tc.input, got, tc.want)
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
		{name: "current month", input: "", want: "2026-10-01"},
		{name: "explicit month", input: "2026-07", want: "2026-07-01"},
		{name: "december", input: "2025-12", want: "2025-12-01"},
		{name: "invalid month", input: "2026-13", err: true},
		{name: "wrong format", input: "oktober", err: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMonth(tc.input, now, loc)
			if tc.err {
				if err == nil {
					t.Fatalf("ParseMonth(%q) wanted an error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMonth(%q) error: %v", tc.input, err)
			}
			if got.Format("2006-01-02") != tc.want {
				t.Fatalf("ParseMonth(%q) = %s, want %s", tc.input, got.Format("2006-01-02"), tc.want)
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
		{name: "comfortable", amount: 100000, spent: 45000, status: StatusOK, remain: 55000, percent: 45},
		{name: "warn at 80 percent", amount: 100000, spent: 80000, status: StatusWarn, remain: 20000, percent: 80},
		{name: "just over", amount: 100000, spent: 100001, status: StatusOver, remain: -1, percent: 100},
		{name: "way over", amount: 200000, spent: 350000, status: StatusOver, remain: -150000, percent: 175},
		{name: "no budget set", amount: 0, spent: 25000, status: StatusNone, remain: -25000, percent: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BudgetStatus(tc.amount, tc.spent); got != tc.status {
				t.Fatalf("BudgetStatus = %s, want %s", got, tc.status)
			}
			if got := BudgetRemaining(tc.amount, tc.spent); got != tc.remain {
				t.Fatalf("BudgetRemaining = %d, want %d", got, tc.remain)
			}
			if got := progress(tc.amount, tc.spent); got != tc.percent {
				t.Fatalf("progress = %d, want %d", got, tc.percent)
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
		{name: "closed month sleeps", locked: true, budget: 100000, spent: 10000, want: moodAsleep},
		{name: "light spending is happy", budget: 100000, spent: 30000, want: moodHappy},
		{name: "exactly 80 percent worries", budget: 100000, spent: 80000, want: moodWorried},
		{name: "a category over budget worries", budget: 500000, spent: 100000, anyOver: true, want: moodWorried},
		{name: "no budget is happy", budget: 0, spent: 250000, want: moodHappy},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Mood(tc.locked, tc.budget, tc.spent, tc.anyOver); got != tc.want {
				t.Fatalf("Mood = %s, want %s", got, tc.want)
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
		{name: "kopi keyword", note: "kopi susu", want: 1},
		{name: "bensin keyword", note: "isi bensin motor", want: 2},
		{name: "listrik keyword", note: "bayar listrik", want: 3},
		{name: "category name", note: "transport harian", want: 2},
		{name: "no match", note: "sesuatu yang aneh", want: 0},
		{name: "blank note", note: "   ", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			match := MatchCategory(tc.note, categories)
			var got int64
			if match != nil {
				got = match.ID
			}
			if got != tc.want {
				t.Fatalf("MatchCategory(%q) = %d, want %d", tc.note, got, tc.want)
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
		{name: "never closed", want: false},
		{
			name:     "closed and not reopened",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			want:     true,
		},
		{
			name:     "reopened after closing",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			reopened: pgtype.Timestamptz{Time: reopenedLater, Valid: true},
			want:     false,
		},
		{
			name:     "closed again after reopening",
			closedAt: pgtype.Timestamptz{Time: closed, Valid: true},
			reopened: pgtype.Timestamptz{Time: reopenedEarlier, Valid: true},
			want:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsLocked(tc.closedAt, tc.reopened); got != tc.want {
				t.Fatalf("IsLocked = %v, want %v", got, tc.want)
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
		{name: "ordinary day", month: time.Date(2026, time.October, 1, 0, 0, 0, 0, loc), day: 5, want: "2026-10-05"},
		{name: "february clamps", month: time.Date(2026, time.February, 1, 0, 0, 0, 0, loc), day: 31, want: "2026-02-28"},
		{name: "april has 30 days", month: time.Date(2026, time.April, 1, 0, 0, 0, 0, loc), day: 31, want: "2026-04-30"},
		{name: "leap year", month: time.Date(2028, time.February, 1, 0, 0, 0, 0, loc), day: 30, want: "2028-02-29"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DueDateFromDay(tc.month, tc.day)
			if got.Format("2006-01-02") != tc.want {
				t.Fatalf("DueDateFromDay = %s, want %s", got.Format("2006-01-02"), tc.want)
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
			t.Fatalf("FormatAmount(%d) = %s, want %s", tc.amount, got, tc.want)
		}
	}
}
