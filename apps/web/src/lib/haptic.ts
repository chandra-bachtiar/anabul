/**
 * Haptic ringan untuk umpan balik saat aksi berhasil. Dipakai kalau perangkat
 * mendukung getaran dan pengguna tidak mematikan animasi di pengaturan sistem.
 */
function bolehGetar(): boolean {
	if (typeof navigator === 'undefined' || typeof navigator.vibrate !== 'function') {
		return false;
	}
	if (typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
		return false;
	}
	return true;
}

/** Getaran singkat, mis. saat catatan berhasil disimpan. */
export function getarRingan(): void {
	if (!bolehGetar()) return;
	navigator.vibrate(12);
}

/** Getaran perayaan, mis. saat hutang lunas atau bulan ditutup. */
export function getarSenang(): void {
	if (!bolehGetar()) return;
	navigator.vibrate([16, 60, 24]);
}

/** Apakah animasi boleh jalan (menghormati pengaturan kurangi gerak). */
export function bolehBergerak(): boolean {
	if (typeof window === 'undefined') return false;
	return !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}
