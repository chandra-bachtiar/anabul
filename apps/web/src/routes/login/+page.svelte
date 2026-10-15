<script lang="ts">
	import { goto } from '$app/navigation';
	import Mascot from '#lib/components/Mascot.svelte';
	import { ApiError } from '#lib/api';
	import { login, session } from '#lib/session.svelte';
	import { toast } from '#lib/toast.svelte';

	let password = $state('');
	let errorMessage = $state('');
	let processing = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (processing) return;

		processing = true;
		errorMessage = '';
		try {
			await login(password);
			if (session.usingDefaultPassword) {
				toast('Password masih default. Sebaiknya diganti di Pengaturan.', 'info', 6000);
			}
			await goto('/');
		} catch (error) {
			errorMessage =
				error instanceof ApiError ? error.message : 'Login gagal. Coba lagi sebentar lagi.';
		} finally {
			processing = false;
		}
	}
</script>

<svelte:head>
	<title>Masuk — ANABUL</title>
</svelte:head>

<div class="flex min-h-[80dvh] flex-col justify-center gap-6">
	<header class="flex flex-col items-center gap-3 text-center">
		<Mascot mood="happy" size={132} />
		<div>
			<h1 class="font-display text-3xl font-semibold tracking-wide text-primary-ink">ANABUL</h1>
			<p class="text-sm opacity-70">Asisten Ngatur Anggaran Bulanan</p>
		</div>
	</header>

	<form class="card bg-base-200" onsubmit={submit}>
		<div class="card-body gap-4">
			<div>
				<label class="label pb-1 text-sm font-semibold" for="password">Password</label>
				<input
					id="password"
					type="password"
					class="input input-bordered min-h-11 w-full text-base"
					bind:value={password}
					autocomplete="current-password"
					required
				/>
			</div>

			{#if errorMessage}
				<p class="rounded-xl border border-error/40 bg-error/10 px-3 py-2 text-sm text-error-ink" role="alert">
					{errorMessage}
				</p>
			{/if}

			{#if !session.configured}
				<p class="rounded-xl border border-warning/50 bg-warning/10 px-3 py-2 text-sm">
					Login belum diaktifkan di server ini. Isi <code>APP_PASSWORD_HASH</code> dan
					<code>SESSION_SECRET</code> lalu jalankan ulang server.
				</p>
			{/if}

			<button type="submit" class="btn btn-primary min-h-11 w-full text-base" disabled={processing}>
				{processing ? 'Memeriksa...' : 'Masuk'}
			</button>

			<p class="text-center text-xs opacity-70">
				Hanya kamu yang pakai aplikasi ini, jadi cukup satu password.
			</p>
		</div>
	</form>
</div>
