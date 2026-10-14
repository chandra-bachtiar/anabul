export interface Toast {
	id: number;
	message: string;
	tone: 'ok' | 'error' | 'info';
}

let counter = 0;

export const toasts = $state<{ items: Toast[] }>({ items: [] });

/**
 * Toast dipakai untuk umpan balik aksi yang tidak perlu memindahkan halaman
 * (simpan budget, bayar kebutuhan, hapus transaksi).
 */
export function toast(message: string, tone: Toast['tone'] = 'ok', duration = 3200): void {
	const id = ++counter;
	toasts.items = [...toasts.items, { id, message, tone }];
	setTimeout(() => dismiss(id), duration);
}

export function dismiss(id: number): void {
	toasts.items = toasts.items.filter((item) => item.id !== id);
}
