package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type intentCase struct {
	Message   string `json:"message"`
	Tool      string `json:"tool"`
	Amount    int64  `json:"amount"`
	Type      string `json:"type"`
	Note      string `json:"note"`
	Person    string `json:"person"`
	Direction string `json:"direction"`
	Category  string `json:"category"`
	// Date: omitted means "not checked", an explicit "" means "must stay empty",
	// "kemarin" is resolved against the fixed test clock, anything else is compared as is.
	Date *string `json:"date"`
}

// testNow keeps the date expectations in testdata stable.
var testNow = time.Date(2026, time.October, 14, 9, 0, 0, 0, time.FixedZone("WIB", 7*60*60))

// TestParseIntent is the message -> tool call regression suite from the PRD examples.
func TestParseIntent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "messages.json"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var cases []intentCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decode testdata: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("testdata is empty")
	}

	for _, tc := range cases {
		t.Run(tc.Message, func(t *testing.T) {
			got := parseIntent(tc.Message, testNow)
			if got.Tool != tc.Tool {
				t.Fatalf("tool = %q, want %q", got.Tool, tc.Tool)
			}
			if tc.Amount != 0 && got.Amount != tc.Amount {
				t.Fatalf("amount = %d, want %d", got.Amount, tc.Amount)
			}
			if tc.Type != "" && got.Type != tc.Type {
				t.Fatalf("type = %q, want %q", got.Type, tc.Type)
			}
			if tc.Note != "" && !strings.EqualFold(got.Note, tc.Note) {
				t.Fatalf("note = %q, want %q", got.Note, tc.Note)
			}
			if tc.Person != "" && !strings.EqualFold(got.Person, tc.Person) {
				t.Fatalf("person = %q, want %q", got.Person, tc.Person)
			}
			if tc.Direction != "" && got.Direction != tc.Direction {
				t.Fatalf("direction = %q, want %q", got.Direction, tc.Direction)
			}
			if tc.Category != "" && !strings.Contains(strings.ToLower(got.Category), tc.Category) {
				t.Fatalf("category = %q, want it to contain %q", got.Category, tc.Category)
			}
			if tc.Date != nil {
				want := *tc.Date
				switch want {
				case "kemarin":
					want = testNow.AddDate(0, 0, -1).Format("2006-01-02")
				case "hari ini":
					want = testNow.Format("2006-01-02")
				}
				if got.Date != want {
					t.Fatalf("date = %q, want %q", got.Date, want)
				}
			}
		})
	}
}

func TestDateFromMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{name: "yesterday", message: "kopi kemarin 25rb", want: "2026-10-13"},
		{name: "today", message: "parkir tadi 5rb", want: "2026-10-14"},
		{name: "day in the current month", message: "belanja tgl 5 40rb", want: "2026-10-05"},
		{name: "a passed day falls back to last month", message: "belanja tanggal 20 150rb", want: "2026-09-20"},
		{name: "a month too short clamps the day", message: "sewa tanggal 31 100rb", want: "2026-09-30"},
		{name: "no time words", message: "kopi 25rb", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIntent(tc.message, testNow)
			if got.Date != tc.want {
				t.Fatalf("date = %q, want %q", got.Date, tc.want)
			}
		})
	}
}

func TestTimeFromDate(t *testing.T) {
	got := timeFromDate("2026-10-13", testNow)
	if got == nil {
		t.Fatal("a valid date must produce a time")
	}
	if got.Format("2006-01-02 15:04") != "2026-10-13 09:00" {
		t.Fatalf("combined time = %s, want 2026-10-13 09:00", got.Format("2006-01-02 15:04"))
	}
	if timeFromDate("", testNow) != nil {
		t.Fatal("an empty date must produce nil")
	}
	if timeFromDate("bukan-tanggal", testNow) != nil {
		t.Fatal("an invalid date must produce nil")
	}
}

func TestRegistryTools(t *testing.T) {
	agent := &Agent{}
	expected := []string{
		"add_transaction", "update_transaction", "delete_transaction", "get_summary",
		"set_budget", "get_budget_remaining", "add_debt", "add_debt_payment", "list_debts",
		"list_monthly_needs", "mark_need_paid", "preview_close_month", "close_month", "reopen_month",
	}

	defs := agent.registry()
	byName := make(map[string]toolDef, len(defs))
	for _, def := range defs {
		byName[def.Name] = def
		if def.Parameters["type"] != "object" {
			t.Fatalf("tool %s: schema must be an object", def.Name)
		}
		if _, ok := def.Parameters["properties"]; !ok {
			t.Fatalf("tool %s: schema without properties", def.Name)
		}
	}

	for _, name := range expected {
		if _, ok := byName[name]; !ok {
			t.Fatalf("tool %s is not registered", name)
		}
	}

	risky := map[string]bool{"delete_transaction": true, "close_month": true, "reopen_month": true}
	for name, def := range byName {
		if def.Risky != risky[name] {
			t.Fatalf("tool %s: risky = %v, want %v", name, def.Risky, risky[name])
		}
	}

	if len(agent.toolSchemas()) != len(defs) {
		t.Fatalf("schema count (%d) does not match tool count (%d)", len(agent.toolSchemas()), len(defs))
	}
}

func TestRiskSummary(t *testing.T) {
	agent := &Agent{tools: nil}
	cases := []struct {
		tool     string
		args     string
		contains string
	}{
		{tool: "delete_transaction", args: `{"id":7}`, contains: "#7"},
		{tool: "delete_transaction", args: `{"last":true}`, contains: "terakhir"},
		{tool: "reopen_month", args: `{"month":"2026-09"}`, contains: "2026-09"},
	}

	for _, tc := range cases {
		t.Run(tc.tool+tc.args, func(t *testing.T) {
			got := agent.riskSummary(tc.tool, json.RawMessage(tc.args))
			if !strings.Contains(got, tc.contains) {
				t.Fatalf("riskSummary = %q, want it to contain %q", got, tc.contains)
			}
		})
	}
}

func TestDescribeResultAndUndo(t *testing.T) {
	describe := describeResult("delete_transaction", json.RawMessage(`{"ok":true,"id":12}`))
	if !strings.Contains(describe, "12") {
		t.Fatalf("describeResult = %q, want it to mention id 12", describe)
	}

	log := []map[string]any{
		{"name": "get_summary", "result": `{"ok":true}`},
		{"name": "add_transaction", "result": `{"ok":true,"id":99,"note":"kopi"}`},
	}
	undo := undoFromLog(log)
	if undo == nil || undo.TransactionID != 99 {
		t.Fatalf("undoFromLog = %+v, want transaction id 99", undo)
	}

	if got := undoFromLog([]map[string]any{{"name": "get_summary", "result": `{"ok":true}`}}); got != nil {
		t.Fatalf("undoFromLog on a read tool = %+v, want nil", got)
	}
}

func TestConfirmationOf(t *testing.T) {
	id, summary, ok := confirmationOf(`{"needs_confirmation":true,"action_id":"abc","summary":"Tutup bulan?"}`)
	if !ok || id != "abc" || summary != "Tutup bulan?" {
		t.Fatalf("confirmationOf wrong: %q %q %v", id, summary, ok)
	}
	if _, _, ok := confirmationOf(`{"ok":true}`); ok {
		t.Fatal("confirmationOf must be false for an ordinary result")
	}
}
