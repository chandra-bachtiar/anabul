export type TransactionType = 'expense' | 'income';
export type DebtDirection = 'hutang' | 'piutang';
export type BudgetStatus = 'ok' | 'warn' | 'over' | 'none';
export type Mood = 'senang' | 'khawatir' | 'tidur';

export interface Transaction {
	id: number;
	type: TransactionType;
	amount: number;
	note: string;
	category_id: number | null;
	category_name?: string;
	account_id: number | null;
	account_name?: string;
	occurred_at: string;
	month: string;
	source: string;
}

export interface Category {
	id: number;
	name: string;
	keywords: string[];
}

export interface Account {
	id: number;
	name: string;
}

export interface BudgetRow {
	category_id: number;
	category_name: string;
	amount: number;
	spent: number;
	remaining: number;
	status: BudgetStatus;
	progress: number;
}

export interface CategorySpend {
	category_id: number | null;
	category_name: string;
	spent: number;
	budget: number;
	remaining: number;
	status: BudgetStatus;
	progress: number;
}

export interface Need {
	id: number;
	name: string;
	amount: number;
	due_date: string;
	status: 'unpaid' | 'paid';
	category_id: number | null;
	category_name?: string;
	transaction_id: number | null;
	overdue: boolean;
	due_soon: boolean;
}

export interface RecurringNeed {
	id: number;
	name: string;
	amount: number;
	due_day: number;
	category_id: number | null;
	category_name?: string;
	active: boolean;
}

export interface DebtPayment {
	id: number;
	amount: number;
	paid_at: string;
}

export interface Debt {
	id: number;
	person: string;
	direction: DebtDirection;
	amount_total: number;
	paid: number;
	remaining: number;
	note: string;
	due_date: string | null;
	status: 'open' | 'settled';
	overdue: boolean;
	created_at: string;
	payments?: DebtPayment[];
}

export interface DebtTotals {
	hutang: number;
	piutang: number;
}

export interface Dashboard {
	month: string;
	month_label: string;
	locked: boolean;
	mood: Mood;
	income: number;
	expense: number;
	net: number;
	budget_total: number;
	budget_spent: number;
	budget_remaining: number;
	budget_status: BudgetStatus;
	budgets: BudgetRow[];
	top_categories: CategorySpend[];
	needs: Need[];
	needs_unpaid: number;
	needs_unpaid_amount: number;
	debts: DebtTotals;
	recent: Transaction[];
	transaction_count: number;
}

export interface Summary {
	month: string;
	month_label: string;
	locked: boolean;
	income: number;
	expense: number;
	net: number;
	budget_total: number;
	budget_spent: number;
	budget_remaining: number;
	budgets: BudgetRow[];
	categories: CategorySpend[];
	unpaid_needs: number;
	unpaid_needs_amount: number;
	debts: DebtTotals;
	top_category: string;
	top_category_amount: number;
	biggest_drain: CategorySpend[];
}

export interface MonthPreview extends Summary {
	unpaid_need_rows: Need[];
	open_debts: Debt[];
	due_debts: Debt[];
	snapshot: Snapshot;
	warnings: string[];
	budget_over: BudgetRow[];
}

export interface Snapshot {
	month: string;
	month_label: string;
	closed_at: string;
	income: number;
	expense: number;
	net: number;
	budget_total: number;
	budget_spent: number;
	budgets: BudgetRow[];
	categories: CategorySpend[];
	unpaid_needs: Need[];
	debts: DebtTotals;
	transaction_count: number;
	carried_budgets: number;
	carried_needs: number;
}

export interface ChatReply {
	reply: string;
	pending_action?: { id: string; summary: string };
	undo?: { transaction_id: number; label?: string };
	source?: string;
}

export interface SessionInfo {
	authenticated: boolean;
	configured?: boolean;
	using_default_password?: boolean;
}

export interface LoginResult {
	ok: boolean;
	expires_at?: string;
	using_default_password?: boolean;
	password_hint?: string;
}

export interface ChangePasswordResult {
	ok: boolean;
	using_default_password?: boolean;
	password_hint?: string;
	min_length?: number;
}
