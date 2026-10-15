import { api, ApiError } from './api';

export interface ChatMessage {
	id: number;
	role: 'user' | 'assistant';
	text: string;
	pending?: { id: string; summary: string; done?: 'confirmed' | 'cancelled' };
	undo?: { transaction_id: number; label?: string; used?: boolean };
	failed?: boolean;
}

let counter = 0;
const listeners = new Set<() => void>();

/** Module-level, so the history survives tab switches within a session. */
export const chatState = $state({
	items: [] as ChatMessage[],
	sending: false
});

export function onChatChange(fn: () => void): () => void {
	listeners.add(fn);
	return () => {
		listeners.delete(fn);
	};
}

function notify(): void {
	for (const fn of listeners) fn();
}

function errorMessage(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.available?.length) {
			return `${error.message}\nPilihan yang ada: ${error.available.join(', ')}`;
		}
		return error.message;
	}
	return 'Ada gangguan saat menghubungi server. Coba lagi ya.';
}

export async function sendMessage(text: string): Promise<void> {
	const content = text.trim();
	if (!content || chatState.sending) return;

	chatState.items.push({ id: ++counter, role: 'user', text: content });
	chatState.sending = true;

	try {
		const reply = await api.chat(content);
		chatState.items.push({
			id: ++counter,
			role: 'assistant',
			text: reply.reply,
			pending: reply.pending_action,
			undo: reply.undo
		});
		notify();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: errorMessage(error), failed: true });
	} finally {
		chatState.sending = false;
	}
}

export async function retryLastMessage(): Promise<void> {
	const last = [...chatState.items].reverse().find((item) => item.role === 'user');
	if (!last) return;
	chatState.items = chatState.items.filter((item) => !item.failed);
	await sendMessage(last.text);
}

export async function confirmAction(message: ChatMessage): Promise<void> {
	if (!message.pending || chatState.sending) return;
	chatState.sending = true;
	try {
		const reply = await api.confirmAction(message.pending.id);
		message.pending.done = 'confirmed';
		chatState.items.push({ id: ++counter, role: 'assistant', text: reply.reply, undo: reply.undo });
		notify();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: errorMessage(error), failed: true });
	} finally {
		chatState.sending = false;
	}
}

export async function cancelAction(message: ChatMessage): Promise<void> {
	if (!message.pending || chatState.sending) return;
	chatState.sending = true;
	try {
		const reply = await api.cancelAction(message.pending.id);
		message.pending.done = 'cancelled';
		chatState.items.push({ id: ++counter, role: 'assistant', text: reply.reply });
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: errorMessage(error), failed: true });
	} finally {
		chatState.sending = false;
	}
}

export async function undoTransaction(message: ChatMessage): Promise<void> {
	if (!message.undo || message.undo.used) return;
	try {
		await api.deleteTransaction(message.undo.transaction_id);
		message.undo.used = true;
		chatState.items.push({ id: ++counter, role: 'assistant', text: 'Sudah dibatalkan, catatannya dihapus.' });
		notify();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: errorMessage(error), failed: true });
	}
}

export function resetChatState(): void {
	chatState.items = [];
}
