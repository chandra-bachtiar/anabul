const rupiah = new Intl.NumberFormat('id-ID', {
	style: 'currency',
	currency: 'IDR',
	maximumFractionDigits: 0
});

const monthsLong = [
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

const monthsShort = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

const weekdaysShort = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];

export function formatRupiah(value: number): string {
	return rupiah.format(value).replace(/\u00a0/g, ' ');
}

/** Drops the "Rp" prefix, for fields that already carry their own label. */
export function formatNumber(value: number): string {
	return formatRupiah(value).replace(/^Rp\s?/, '');
}

/** Narrow-space form for badges and progress labels: "1,5jt", "250rb". */
export function formatCompact(value: number): string {
	const abs = Math.abs(value);
	if (abs >= 1_000_000) {
		const millions = value / 1_000_000;
		return `${millions.toFixed(millions % 1 === 0 ? 0 : 1).replace('.', ',')}jt`;
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

export function formatMonth(month: string): string {
	const parsed = parts(month);
	if (!parsed) return month;
	return `${monthsLong[parsed.month - 1]} ${parsed.year}`;
}

export function formatMonthShort(month: string): string {
	const parsed = parts(month);
	if (!parsed) return month;
	return `${monthsShort[parsed.month - 1]} ${parsed.year}`;
}

export function currentMonth(): string {
	const now = new Date();
	return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

export function shiftMonth(month: string, delta: number): string {
	const parsed = parts(month);
	if (!parsed) return currentMonthFallback();
	const date = new Date(parsed.year, parsed.month - 1 + delta, 1);
	return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

function currentMonthFallback(): string {
	return currentMonth();
}

export function formatDate(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return `${date.getDate()} ${monthsShort[date.getMonth()]} ${date.getFullYear()}`;
}

export function formatTime(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return '';
	return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

export function formatDay(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;

	const today = new Date();
	const startOf = (value: Date) => new Date(value.getFullYear(), value.getMonth(), value.getDate()).getTime();
	const diff = Math.round((startOf(today) - startOf(date)) / 86_400_000);

	if (diff === 0) return 'Hari ini';
	if (diff === 1) return 'Kemarin';
	return `${weekdaysShort[date.getDay()]}, ${date.getDate()} ${monthsShort[date.getMonth()]}`;
}

/** Short due-date form ("5 Okt"); a need without a due date yields "". */
export function formatDateShort(iso: string | null): string {
	if (!iso) return '';
	const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
	if (!match) return iso;
	return `${Number(match[3])} ${monthsShort[Number(match[2]) - 1]}`;
}

/** Local "YYYY-MM-DD" for input type="date"; defaults to today. */
export function toDateInputValue(value?: string | Date): string {
	const date = value === undefined ? new Date() : value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const day = String(date.getDate()).padStart(2, '0');
	return `${date.getFullYear()}-${month}-${day}`;
}

export function yesterdayInputValue(): string {
	return toDateInputValue(new Date(Date.now() - 86_400_000));
}

/**
 * Rebuilds an RFC3339 timestamp from a "YYYY-MM-DD" input, reusing the clock
 * time of an existing value so edits do not reset the hour.
 */
export function isoFromDateInput(date: string, timeSource?: string): string {
	const parsed = timeSource ? new Date(timeSource) : new Date();
	const clock = Number.isNaN(parsed.getTime()) ? new Date() : parsed;
	const hour = String(clock.getHours()).padStart(2, '0');
	const minute = String(clock.getMinutes()).padStart(2, '0');
	const second = String(clock.getSeconds()).padStart(2, '0');
	return new Date(`${date}T${hour}:${minute}:${second}`).toISOString();
}

export function monthFromDateInput(date: string): string {
	return date.slice(0, 7);
}

export function budgetStatusLabel(status: string): string {
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
