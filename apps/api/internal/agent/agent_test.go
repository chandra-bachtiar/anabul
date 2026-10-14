package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
}

// TestParseIntent adalah regresi pesan -> tool call memakai contoh di PRD.
func TestParseIntent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "messages.json"))
	if err != nil {
		t.Fatalf("baca testdata: %v", err)
	}
	var cases []intentCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decode testdata: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("testdata kosong")
	}

	for _, tc := range cases {
		t.Run(tc.Message, func(t *testing.T) {
			got := parseIntent(tc.Message)
			if got.Tool != tc.Tool {
				t.Fatalf("tool = %q, ingin %q", got.Tool, tc.Tool)
			}
			if tc.Amount != 0 && got.Amount != tc.Amount {
				t.Fatalf("amount = %d, ingin %d", got.Amount, tc.Amount)
			}
			if tc.Type != "" && got.Type != tc.Type {
				t.Fatalf("type = %q, ingin %q", got.Type, tc.Type)
			}
			if tc.Note != "" && !strings.EqualFold(got.Note, tc.Note) {
				t.Fatalf("note = %q, ingin %q", got.Note, tc.Note)
			}
			if tc.Person != "" && !strings.EqualFold(got.Person, tc.Person) {
				t.Fatalf("person = %q, ingin %q", got.Person, tc.Person)
			}
			if tc.Direction != "" && got.Direction != tc.Direction {
				t.Fatalf("direction = %q, ingin %q", got.Direction, tc.Direction)
			}
			if tc.Category != "" && !strings.Contains(strings.ToLower(got.Category), tc.Category) {
				t.Fatalf("category = %q, ingin mengandung %q", got.Category, tc.Category)
			}
		})
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
			t.Fatalf("tool %s: schema harus object", def.Name)
		}
		if _, ok := def.Parameters["properties"]; !ok {
			t.Fatalf("tool %s: schema tanpa properties", def.Name)
		}
	}

	for _, name := range expected {
		if _, ok := byName[name]; !ok {
			t.Fatalf("tool %s tidak terdaftar", name)
		}
	}

	risky := map[string]bool{"delete_transaction": true, "close_month": true, "reopen_month": true}
	for name, def := range byName {
		if def.Risky != risky[name] {
			t.Fatalf("tool %s: risky = %v, ingin %v", name, def.Risky, risky[name])
		}
	}

	if len(agent.toolSchemas()) != len(defs) {
		t.Fatalf("jumlah schema (%d) tidak sama dengan jumlah tool (%d)", len(agent.toolSchemas()), len(defs))
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
				t.Fatalf("riskSummary = %q, ingin mengandung %q", got, tc.contains)
			}
		})
	}
}

func TestDescribeResultAndUndo(t *testing.T) {
	describe := describeResult("delete_transaction", json.RawMessage(`{"ok":true,"id":12}`))
	if !strings.Contains(describe, "12") {
		t.Fatalf("describeResult = %q, ingin menyebut id 12", describe)
	}

	log := []map[string]any{
		{"name": "get_summary", "result": `{"ok":true}`},
		{"name": "add_transaction", "result": `{"ok":true,"id":99,"note":"kopi"}`},
	}
	undo := undoFromLog(log)
	if undo == nil || undo.TransactionID != 99 {
		t.Fatalf("undoFromLog = %+v, ingin transaction id 99", undo)
	}

	if got := undoFromLog([]map[string]any{{"name": "get_summary", "result": `{"ok":true}`}}); got != nil {
		t.Fatalf("undoFromLog untuk tool baca = %+v, ingin nil", got)
	}
}

func TestConfirmationOf(t *testing.T) {
	id, summary, ok := confirmationOf(`{"needs_confirmation":true,"action_id":"abc","summary":"Tutup bulan?"}`)
	if !ok || id != "abc" || summary != "Tutup bulan?" {
		t.Fatalf("confirmationOf salah: %q %q %v", id, summary, ok)
	}
	if _, _, ok := confirmationOf(`{"ok":true}`); ok {
		t.Fatal("confirmationOf harus false untuk hasil biasa")
	}
}
