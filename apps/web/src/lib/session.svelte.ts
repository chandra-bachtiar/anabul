import { api, ApiError } from './api';

/**
 * A single reactive object, so layout, pages, and the login/logout/change
 * password actions all read the same session.
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
		const result = await api.login(password);
		session.authenticated = true;
		session.usingDefaultPassword = result.using_default_password ?? false;
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
