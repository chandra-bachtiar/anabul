type ThemeMode = 'system' | 'light' | 'dark';

/** Key stays as-is: renaming it would wipe the theme saved on existing devices. */
const STORAGE_KEY = 'anabul-tema';

/**
 * Theme follows the system unless the user forces light or dark in Settings;
 * the choice is stored per device.
 */
export const theme = $state({
	mode: 'system' as ThemeMode,
	resolved: 'light' as 'light' | 'dark'
});

function prefersDark(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

export function applyTheme(): void {
	if (typeof document === 'undefined') return;
	const root = document.documentElement;

	if (theme.mode === 'system') {
		root.removeAttribute('data-theme');
		theme.resolved = prefersDark() ? 'dark' : 'light';
	} else {
		root.setAttribute('data-theme', theme.mode === 'dark' ? 'anabul-dark' : 'anabul');
		theme.resolved = theme.mode;
	}
}

export function setTheme(mode: ThemeMode): void {
	theme.mode = mode;
	if (typeof localStorage !== 'undefined') {
		localStorage.setItem(STORAGE_KEY, mode);
	}
	applyTheme();
}

export function initTheme(): void {
	if (typeof localStorage !== 'undefined') {
		const saved = localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
		if (saved === 'light' || saved === 'dark' || saved === 'system') {
			theme.mode = saved;
		}
	}
	applyTheme();

	if (typeof window !== 'undefined') {
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
			if (theme.mode === 'system') applyTheme();
		});
	}
}
