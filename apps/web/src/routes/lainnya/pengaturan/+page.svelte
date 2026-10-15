<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError, api } from '#lib/api';
	import Mascot from '#lib/components/Mascot.svelte';
	import { logout, session } from '#lib/session.svelte';
	import { setTheme, theme } from '#lib/theme.svelte';
	import { toast } from '#lib/toast.svelte';

	let signingOut = $state(false);

	let currentPassword = $state('');
	let newPassword = $state('');
	let repeatPassword = $state('');
	let passwordError = $state('');
	let savingPassword = $state(false);

	async function signOut() {
		signingOut = true;
		try {
			await logout();
			await goto('/login');
		} catch {
			toast('Gagal keluar, coba lagi ya.', 'error');
		} finally {
			signingOut = false;
		}
	}

	async function changePassword(event: SubmitEvent) {
		event.preventDefault();
		if (savingPassword) return;

		passwordError = '';
		if (newPassword.length < 8) {
			passwordError = 'Password baru minimal 8 karakter.';
			return;
		}
		if (newPassword !== repeatPassword) {
			passwordError = 'Ulangi password baru belum sama.';
			return;
		}

		savingPassword = true;
		try {
			const result = await api.changePassword(currentPassword, newPassword);
			session.usingDefaultPassword = result.using_default_password ?? false;
			currentPassword = '';
			newPassword = '';
			repeatPassword = '';
			toast('Password diganti. Catat yang baru ya.', 'ok', 5000);
		} catch (error) {
			passwordError =
				error instanceof ApiError ? error.message : 'Gagal mengganti password, coba lagi.';
		} finally {
			savingPassword = false;
		}
	}

	const themeOptions = [
		{ value: 'system' as const, label: 'Ikut sistem' },
		{ value: 'light' as const, label: 'Terang' },
		{ value: 'dark' as const, label: 'Gelap' }
	];
</script>

<svelte:head>
	<title>Pengaturan — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Pengaturan</h1>

	<section class="card bg-base-200" aria-labelledby="judul-tema">
		<div class="card-body gap-3 p-4">
			<h2 id="judul-tema" class="font-display text-base font-semibold">Tema</h2>
			<p class="text-xs opacity-70">Krem hangat untuk siang, cokelat gelap untuk malam. Pilihanmu disimpan di perangkat ini.</p>

			<div class="grid grid-cols-3 gap-2" role="radiogroup" aria-label="Pilih tema">
				{#each themeOptions as item (item.value)}
					<button
						type="button"
						role="radio"
						aria-checked={theme.mode === item.value}
						class="btn min-h-11 {theme.mode === item.value ? 'btn-primary' : 'btn-outline'}"
						onclick={() => setTheme(item.value)}
					>
						{item.label}
					</button>
				{/each}
			</div>
		</div>
	</section>

	<section id="ganti-password" class="card bg-base-200" aria-labelledby="judul-password">
		<div class="card-body gap-3 p-4">
			<div>
				<h2 id="judul-password" class="font-display text-base font-semibold">Ganti password</h2>
				<p class="mt-1 text-xs opacity-70">
					Minimal 8 karakter. Password baru disimpan di database, jadi tidak perlu ubah environment lagi.
				</p>
			</div>

			{#if session.usingDefaultPassword}
				<p class="rounded-xl border border-warning/60 bg-warning/10 px-3 py-2 text-sm">
					Password sekarang masih <strong>anabul123</strong> (default dari README). Sebaiknya ganti sekarang.
				</p>
			{/if}

			<form class="space-y-3" onsubmit={changePassword}>
				<div>
					<label class="label pb-1 text-sm font-semibold" for="password-lama">Password sekarang</label>
					<input
						id="password-lama"
						type="password"
						autocomplete="current-password"
						class="input input-bordered min-h-11 w-full"
						bind:value={currentPassword}
						required
					/>
				</div>
				<div>
					<label class="label pb-1 text-sm font-semibold" for="password-baru">Password baru</label>
					<input
						id="password-baru"
						type="password"
						autocomplete="new-password"
						minlength="8"
						class="input input-bordered min-h-11 w-full"
						bind:value={newPassword}
						required
					/>
				</div>
				<div>
					<label class="label pb-1 text-sm font-semibold" for="password-ulang">Ulangi password baru</label>
					<input
						id="password-ulang"
						type="password"
						autocomplete="new-password"
						minlength="8"
						class="input input-bordered min-h-11 w-full"
						bind:value={repeatPassword}
						required
					/>
				</div>

				{#if passwordError}
					<p class="rounded-xl border border-error/40 bg-error/10 px-3 py-2 text-sm text-error-ink" role="alert">
						{passwordError}
					</p>
				{/if}

				<button type="submit" class="btn btn-primary min-h-11 self-start" disabled={savingPassword}>
					{savingPassword ? 'Menyimpan...' : 'Simpan password baru'}
				</button>
			</form>
		</div>
	</section>

	<section class="card bg-base-200" aria-labelledby="judul-tentang">
		<div class="card-body flex-row items-start gap-3 p-4">
			<Mascot mood="happy" size={56} alt="" />
			<div class="space-y-1">
				<h2 id="judul-tentang" class="font-display text-base font-semibold">Tentang ANABUL</h2>
				<p class="text-sm opacity-80">
					Asisten Ngatur Anggaran Bulanan. Catatan, budget, kebutuhan, dan hutang piutang disimpan di
					Postgres milikmu sendiri.
				</p>
				<p class="text-xs opacity-70">
					Agent LLM hanya memilih tool dan mengisi argumen. Semua perhitungan dan validasi dikerjakan
					server.
				</p>
			</div>
		</div>
	</section>

	<section class="card bg-base-200" aria-labelledby="judul-keluar">
		<div class="card-body gap-3 p-4">
			<h2 id="judul-keluar" class="font-display text-base font-semibold">Sesi</h2>
			<p class="text-sm opacity-80">Keluar akan menghapus cookie sesi di perangkat ini. Datamu tetap aman di server.</p>
			<button type="button" class="btn btn-outline min-h-11 self-start" onclick={signOut} disabled={signingOut}>
				{signingOut ? 'Keluar...' : 'Keluar dari sesi'}
			</button>
		</div>
	</section>
</div>
