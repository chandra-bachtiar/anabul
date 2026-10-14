import { api, ApiError } from './api';

/**
 * Sesi disimpan di satu objek reaktif supaya layout, halaman, dan aksi
 * (login, logout, ganti password) membaca state yang sama.
 */
export const session = $state({
	checked: false,
	authenticated: false,
	configured: true,
	usingDefaultPassword: false,
	error: '' as string
});

export async function checkSession(): Promise<void> {
	try {
		const info = await api.me();
		session.authenticated = info.authenticated;
		session.configured = info.configured ?? true;
		session.usingDefaultPassword = info.using_default_password ?? false;
		session.error = '';
	} catch (error) {
		session.authenticated = false;
		session.usingDefaultPassword = false;
		session.error = error instanceof Error ? error.message : 'Gagal memeriksa sesi';
	} finally {
		session.checked = true;
	}
}

export async function login(password: string): Promise<void> {
	try {
		const hasil = await api.login(password);
		session.authenticated = true;
		session.usingDefaultPassword = hasil.using_default_password ?? false;
		session.error = '';
	} catch (error) {
		throw new ApiError(
			error instanceof ApiError ? error.message : 'Login gagal, coba lagi ya',
			error instanceof ApiError ? error.status : 0
		);
	}
}

export async function logout(): Promise<void> {
	try {
		await api.logout();
	} finally {
		session.authenticated = false;
		session.usingDefaultPassword = false;
	}
}
