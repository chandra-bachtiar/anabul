const rupiah = new Intl.NumberFormat('id-ID', {
	style: 'currency',
	currency: 'IDR',
	maximumFractionDigits: 0
});

const bulanPanjang = [
	'Januari',
	'Februari',
	'Maret',
	'April',
	'Mei',
	'Juni',
	'Juli',
	'Agustus',
	'September',
	'Oktober',
	'November',
	'Desember'
];

const bulanPendek = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

const hariPendek = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];

/** Rupiah penuh untuk nominal yang perlu dibaca persis: "Rp 25.000". */
export function formatRupiah(value: number): string {
	return rupiah.format(value).replace(/\u00a0/g, ' ');
}

/** Tanpa awalan "Rp", untuk kolom angka yang sudah punya label. */
export function formatAngka(value: number): string {
	return formatRupiah(value).replace(/^Rp\s?/, '');
}

/**
 * Bentuk pendek untuk ruang sempit (badge, label progress): "1,5jt", "250rb".
 * Dipakai hanya kalau lebar tidak cukup untuk nominal penuh.
 */
export function formatRingkas(value: number): string {
	const abs = Math.abs(value);
	if (abs >= 1_000_000) {
		const juta = value / 1_000_000;
		return `${juta.toFixed(juta % 1 === 0 ? 0 : 1).replace('.', ',')}jt`;
	}
	if (abs >= 1_000) {
		return `${Math.round(value / 1_000)}rb`;
	}
	return String(value);
}

function parts(month: string): { year: number; month: number } | null {
	const match = /^(\d{4})-(\d{2})/.exec(month);
	if (!match) return null;
	return { year: Number(match[1]), month: Number(match[2]) };
}

/** "2026-10" -> "Oktober 2026" */
export function labelBulan(month: string): string {
	const parsed = parts(month);
	if (!parsed) return month;
	return `${bulanPanjang[parsed.month - 1]} ${parsed.year}`;
}

/** "2026-10" -> "Okt 2026", untuk tab dan chip. */
export function labelBulanPendek(month: string): string {
	const parsed = parts(month);
	if (!parsed) return month;
	return `${bulanPendek[parsed.month - 1]} ${parsed.year}`;
}

export function bulanIni(): string {
	const now = new Date();
	return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

export function geserBulan(month: string, delta: number): string {
	const parsed = parts(month);
	if (!parsed) return monthIniFallback();
	const date = new Date(parsed.year, parsed.month - 1 + delta, 1);
	return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

function monthIniFallback(): string {
	return bulanIni();
}

/** Tanggal panjang: "13 Okt 2026" */
export function labelTanggal(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return `${date.getDate()} ${bulanPendek[date.getMonth()]} ${date.getFullYear()}`;
}

/** Jam: "20:15" */
export function labelJam(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return '';
	return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

/** Judul kelompok di daftar transaksi: "Hari ini", "Kemarin", atau "Sen, 13 Okt". */
export function labelHari(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;

	const today = new Date();
	const startOf = (value: Date) => new Date(value.getFullYear(), value.getMonth(), value.getDate()).getTime();
	const diff = Math.round((startOf(today) - startOf(date)) / 86_400_000);

	if (diff === 0) return 'Hari ini';
	if (diff === 1) return 'Kemarin';
	return `${hariPendek[date.getDay()]}, ${date.getDate()} ${bulanPendek[date.getMonth()]}`;
}

/** "2026-10-05" -> "5 Okt" (dipakai untuk jatuh tempo). */
export function labelTanggalPendek(iso: string | null): string {
	if (!iso) return '';
	const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
	if (!match) return iso;
	return `${Number(match[3])} ${bulanPendek[Number(match[2]) - 1]}`;
}

export function statusBudgetLabel(status: string): string {
	switch (status) {
		case 'ok':
			return 'Aman, meong~';
		case 'warn':
			return 'Hampir habis, hati-hati ya';
		case 'over':
			return 'Sudah lewat';
		default:
			return 'Belum diatur';
	}
}
