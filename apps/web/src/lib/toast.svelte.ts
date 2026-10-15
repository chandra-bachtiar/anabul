export interface Toast {
	id: number;
	message: string;
	tone: 'ok' | 'error' | 'info';
}

let counter = 0;

export const toasts = $state<{ items: Toast[] }>({ items: [] });

/**
 * Toasts carry feedback for actions that must not move the user off the page:
 * saving a budget, paying a need, deleting a transaction.
 */
export function toast(message: string, tone: Toast['tone'] = 'ok', duration = 3200): void {
	const id = ++counter;
	toasts.items = [...toasts.items, { id, message, tone }];
	setTimeout(() => dismiss(id), duration);
}

export function dismiss(id: number): void {
	toasts.items = toasts.items.filter((item) => item.id !== id);
}
