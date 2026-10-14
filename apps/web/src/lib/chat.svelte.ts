import { api, ApiError } from './api';

export interface PesanChat {
	id: number;
	role: 'user' | 'assistant';
	text: string;
	pending?: { id: string; summary: string; selesai?: 'dikonfirmasi' | 'dibatalkan' };
	undo?: { transaction_id: number; label?: string; dipakai?: boolean };
	gagal?: boolean;
}

let counter = 0;
const listeners = new Set<() => void>();

/**
 * Riwayat chat disimpan di level modul supaya tidak hilang saat berpindah tab
 * dalam satu sesi berjalan.
 */
export const chatState = $state({
	items: [] as PesanChat[],
	mengirim: false
});

export function onChatChange(fn: () => void): () => void {
	listeners.add(fn);
	return () => {
		listeners.delete(fn);
	};
}

function beriTahu(): void {
	for (const fn of listeners) fn();
}

function pesanError(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.available?.length) {
			return `${error.message}\nPilihan yang ada: ${error.available.join(', ')}`;
		}
		return error.message;
	}
	return 'Ada gangguan saat menghubungi server. Coba lagi ya.';
}

export async function kirimPesan(teks: string): Promise<void> {
	const isi = teks.trim();
	if (!isi || chatState.mengirim) return;

	chatState.items.push({ id: ++counter, role: 'user', text: isi });
	chatState.mengirim = true;

	try {
		const balasan = await api.chat(isi);
		chatState.items.push({
			id: ++counter,
			role: 'assistant',
			text: balasan.reply,
			pending: balasan.pending_action,
			undo: balasan.undo
		});
		beriTahu();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: pesanError(error), gagal: true });
	} finally {
		chatState.mengirim = false;
	}
}

export async function ulangiPesanTerakhir(): Promise<void> {
	const terakhir = [...chatState.items].reverse().find((item) => item.role === 'user');
	if (!terakhir) return;
	chatState.items = chatState.items.filter((item) => !item.gagal);
	await kirimPesan(terakhir.text);
}

export async function konfirmasiAksi(pesan: PesanChat): Promise<void> {
	if (!pesan.pending || chatState.mengirim) return;
	chatState.mengirim = true;
	try {
		const balasan = await api.confirmAction(pesan.pending.id);
		pesan.pending.selesai = 'dikonfirmasi';
		chatState.items.push({ id: ++counter, role: 'assistant', text: balasan.reply, undo: balasan.undo });
		beriTahu();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: pesanError(error), gagal: true });
	} finally {
		chatState.mengirim = false;
	}
}

export async function batalAksi(pesan: PesanChat): Promise<void> {
	if (!pesan.pending || chatState.mengirim) return;
	chatState.mengirim = true;
	try {
		const balasan = await api.cancelAction(pesan.pending.id);
		pesan.pending.selesai = 'dibatalkan';
		chatState.items.push({ id: ++counter, role: 'assistant', text: balasan.reply });
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: pesanError(error), gagal: true });
	} finally {
		chatState.mengirim = false;
	}
}

export async function batalkanTransaksi(pesan: PesanChat): Promise<void> {
	if (!pesan.undo || pesan.undo.dipakai) return;
	try {
		await api.deleteTransaction(pesan.undo.transaction_id);
		pesan.undo.dipakai = true;
		chatState.items.push({ id: ++counter, role: 'assistant', text: 'Sudah dibatalkan, catatannya dihapus.' });
		beriTahu();
	} catch (error) {
		chatState.items.push({ id: ++counter, role: 'assistant', text: pesanError(error), gagal: true });
	}
}

export function resetChat(): void {
	chatState.items = [];
}
