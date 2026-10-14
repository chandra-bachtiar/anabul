import type {
	Account,
	BudgetRow,
	Category,
	ChangePasswordResult,
	ChatReply,
	Dashboard,
	Debt,
	DebtTotals,
	LoginResult,
	MonthPreview,
	Need,
	RecurringNeed,
	SessionInfo,
	Snapshot,
	Summary,
	Transaction
} from './types';

export class ApiError extends Error {
	status: number;
	code?: string;
	available?: string[];

	constructor(message: string, status: number, code?: string, available?: string[]) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
		this.available = available;
	}
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	let response: Response;
	try {
		response = await fetch(path, {
			method,
			credentials: 'same-origin',
			headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	} catch {
		throw new ApiError('Tidak bisa menghubungi server. Cek koneksimu lalu coba lagi.', 0);
	}

	const text = await response.text();
	let payload: unknown = null;
	if (text) {
		try {
			payload = JSON.parse(text);
		} catch {
			payload = null;
		}
	}

	if (!response.ok) {
		const data = (payload ?? {}) as { error?: string; code?: string; available?: string[] };
		throw new ApiError(data.error ?? 'Ada gangguan di server, coba lagi ya.', response.status, data.code, data.available);
	}

	return payload as T;
}

function query(params: Record<string, string | number | undefined | null>): string {
	const search = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value === undefined || value === null || value === '') continue;
		search.set(key, String(value));
	}
	const encoded = search.toString();
	return encoded ? `?${encoded}` : '';
}

export const api = {
	// Auth
	me: () => request<SessionInfo>('GET', '/api/auth/me'),
	login: (password: string) => request<LoginResult>('POST', '/api/auth/login', { password }),
	logout: () => request<{ ok: boolean }>('POST', '/api/auth/logout'),
	changePassword: (current_password: string, new_password: string) =>
		request<ChangePasswordResult>('POST', '/api/auth/password', { current_password, new_password }),

	// Home
	dashboard: (month?: string) => request<Dashboard>('GET', `/api/dashboard${query({ month })}`),
	summary: (month?: string) => request<Summary>('GET', `/api/summary${query({ month })}`),

	// Transaksi
	transactions: (params: { month?: string; category_id?: number; q?: string; cursor?: number; limit?: number }) =>
		request<{ transactions: Transaction[]; next_cursor: number | null }>('GET', `/api/transactions${query(params)}`),
	createTransaction: (payload: {
		type: string;
		amount: number;
		note?: string;
		category_id?: number | null;
		account_id?: number | null;
		occurred_at?: string;
	}) => request<Transaction>('POST', '/api/transactions', payload),
	updateTransaction: (id: number, payload: Record<string, unknown>) =>
		request<Transaction>('PATCH', `/api/transactions/${id}`, payload),
	deleteTransaction: (id: number) => request<{ ok: boolean }>('DELETE', `/api/transactions/${id}`),

	// Kategori dan akun
	categories: () => request<{ categories: Category[] }>('GET', '/api/categories'),
	createCategory: (name: string, keywords: string[] = []) =>
		request<Category>('POST', '/api/categories', { name, keywords }),
	updateCategory: (id: number, payload: { name?: string; keywords?: string[] }) =>
		request<Category>('PATCH', `/api/categories/${id}`, payload),
	deleteCategory: (id: number) => request<{ ok: boolean }>('DELETE', `/api/categories/${id}`),
	accounts: () => request<{ accounts: Account[] }>('GET', '/api/accounts'),
	createAccount: (name: string) => request<Account>('POST', '/api/accounts', { name }),

	// Budget
	budgets: (month?: string) => request<{ budgets: BudgetRow[] }>('GET', `/api/budgets${query({ month })}`),
	saveBudgets: (month: string, items: { category_id: number; amount: number }[]) =>
		request<{ budgets: BudgetRow[] }>('PUT', '/api/budgets', { month, items }),
	copyBudgets: (from: string, to: string) =>
		request<{ ok: boolean; copied: number }>('POST', '/api/budgets/copy', { from, to }),

	// Kebutuhan bulanan
	recurringNeeds: () => request<{ recurring_needs: RecurringNeed[] }>('GET', '/api/recurring-needs'),
	createRecurringNeed: (payload: { name: string; amount: number; due_day: number; category_id?: number | null }) =>
		request<RecurringNeed>('POST', '/api/recurring-needs', payload),
	updateRecurringNeed: (id: number, payload: Record<string, unknown>) =>
		request<RecurringNeed>('PATCH', `/api/recurring-needs/${id}`, payload),
	deleteRecurringNeed: (id: number) => request<{ ok: boolean }>('DELETE', `/api/recurring-needs/${id}`),
	needs: (month?: string, status?: string) =>
		request<{ needs: Need[] }>('GET', `/api/needs${query({ month, status })}`),
	generateNeeds: (month?: string) => request<{ ok: boolean; needs: number }>('POST', '/api/needs/generate', { month }),
	payNeed: (id: number, amount?: number) => request<Need>('POST', `/api/needs/${id}/pay`, amount ? { amount } : {}),
	unpayNeed: (id: number) => request<Need>('POST', `/api/needs/${id}/unpay`, {}),

	// Hutang piutang
	debts: (status?: string, person?: string) =>
		request<{ debts: Debt[]; totals: DebtTotals }>('GET', `/api/debts${query({ status, person })}`),
	createDebt: (payload: {
		person: string;
		direction: string;
		amount: number;
		note?: string;
		due_date?: string | null;
	}) => request<Debt>('POST', '/api/debts', payload),
	debt: (id: number) => request<Debt>('GET', `/api/debts/${id}`),
	createDebtPayment: (id: number, amount: number, paid_at?: string) =>
		request<Debt>('POST', `/api/debts/${id}/payments`, { amount, paid_at }),

	// Tutup bulan
	months: () => request<{ closed_months: string[]; current_month: string }>('GET', '/api/months'),
	monthPreview: (month: string) => request<MonthPreview>('GET', `/api/months/${month}/preview`),
	closeMonth: (month: string) => request<Snapshot>('POST', `/api/months/${month}/close`, { confirm: true }),
	reopenMonth: (month: string) => request<{ ok: boolean }>('POST', `/api/months/${month}/reopen`, { confirm: true }),

	// Agent
	chat: (message: string) => request<ChatReply>('POST', '/api/chat', { message }),
	confirmAction: (id: string) => request<ChatReply>('POST', `/api/actions/${id}/confirm`, {}),
	cancelAction: (id: string) => request<ChatReply>('POST', `/api/actions/${id}/cancel`, {})
};
