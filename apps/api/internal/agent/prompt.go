package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"anabul/internal/tools"
)

// systemPrompt rebuilds the context sent to the model on every request.
func (a *Agent) systemPrompt(ctx context.Context, channel string) string {
	var b strings.Builder
	now := a.tools.Now()

	b.WriteString("You are ANABUL, a friendly cat assistant for the owner's personal finance.\n")
	b.WriteString("Always answer in casual Indonesian (bahasa gaul, kamu/gua boleh santai), max one emoji, an occasional \"meong\" is fine.\n")
	b.WriteString("Amounts and statuses stay explicit. Drop the jokes for errors, risky-action confirmations, and month closing.\n")
	b.WriteString("If something important is missing (an amount, for example), ask one short question back instead of guessing.\n\n")

	fmt.Fprintf(&b, "Today is %s (%s), current month %s.\n", now.Format("2006-01-02"), a.tools.Location().String(), tools.FormatMonth(a.tools.CurrentMonth()))
	fmt.Fprintf(&b, "Conversation channel: %s.\n", channel)

	if categories, err := a.tools.ListCategories(ctx); err == nil && len(categories) > 0 {
		names := make([]string, 0, len(categories))
		for _, category := range categories {
			names = append(names, category.Name)
		}
		fmt.Fprintf(&b, "Available categories: %s.\n", strings.Join(names, ", "))
	}
	if accounts, err := a.tools.ListAccounts(ctx); err == nil && len(accounts) > 0 {
		names := make([]string, 0, len(accounts))
		for _, account := range accounts {
			names = append(names, account.Name)
		}
		fmt.Fprintf(&b, "Available accounts: %s.\n", strings.Join(names, ", "))
	}

	month := a.tools.CurrentMonth()
	if isLocked, err := a.tools.IsLocked(ctx, month); err == nil {
		if isLocked {
			fmt.Fprintf(&b, "Month %s is LOCKED: every write is rejected, offer to reopen it through the confirmation flow.\n", tools.FormatMonth(month))
		} else {
			fmt.Fprintf(&b, "Month %s is open.\n", tools.FormatMonth(month))
		}
	}

	if needs, err := a.tools.ListMonthlyNeeds(ctx, tools.FormatMonth(month), "unpaid"); err == nil && len(needs) > 0 {
		items := make([]string, 0, len(needs))
		for _, need := range needs {
			items = append(items, fmt.Sprintf("%s %s (%s)", need.Name, tools.FormatAmount(need.Amount), dueLabel(need)))
		}
		fmt.Fprintf(&b, "Unpaid needs this month: %s.\n", strings.Join(items, "; "))
	}

	if debts, err := a.tools.ListDebts(ctx, "open", ""); err == nil && len(debts) > 0 {
		items := make([]string, 0, len(debts))
		for _, debt := range debts {
			items = append(items, fmt.Sprintf("%s %s %s (id %d)", debt.Direction, debt.Person, tools.FormatAmount(debt.Remaining), debt.ID))
		}
		fmt.Fprintf(&b, "Open debts and receivables: %s.\n", strings.Join(items, "; "))
	}

	if summary, err := a.tools.MonthSummary(ctx, ""); err == nil {
		fmt.Fprintf(&b, "Month so far: income %s, expense %s, budget left %s.\n",
			tools.FormatAmount(summary.Income), tools.FormatAmount(summary.Expense), tools.FormatAmount(summary.BudgetLeft))
	}

	b.WriteString("\nRules: call a tool for every data change. Risky actions (deleting a transaction, closing or reopening a month) are never executed inline; the tool answers with needs_confirmation and the user decides.\n")
	b.WriteString("Amounts are integer rupiah without separators.\n")
	b.WriteString("When the user names another day (\"kemarin\", \"tgl 5\"), fill the date argument as YYYY-MM-DD relative to today's date above. Leave it empty to record today.\n")
	return b.String()
}

func dueLabel(need tools.NeedRow) string {
	switch {
	case need.Overdue:
		return "overdue " + need.DueDate
	case need.DueSoon:
		return "due soon " + need.DueDate
	default:
		return "due " + need.DueDate
	}
}

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
