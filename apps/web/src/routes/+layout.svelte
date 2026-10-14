<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import BottomNav from '#lib/components/BottomNav.svelte';
	import Toasts from '#lib/components/Toasts.svelte';
	import { bolehBergerak } from '#lib/haptic';
	import { checkSession, session } from '#lib/session.svelte';
	import { initTheme } from '#lib/theme.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const jalur = $derived(page.url.pathname);
	const diLogin = $derived(jalur === '/login');
	const durasiPindah = $derived(bolehBergerak() ? 180 : 0);

	onMount(() => {
		initTheme();
		void checkSession();

		// Service worker hanya di build produksi; saat dev bisa mengganggu HMR.
		if (import.meta.env.PROD && 'serviceWorker' in navigator) {
			navigator.serviceWorker.register('/sw.js').catch(() => {
				// Offline mode gagal disiapkan, aplikasi tetap jalan online.
			});
		}
	});

	$effect(() => {
		if (session.checked && !session.authenticated && !diLogin) {
			void goto('/login', { replaceState: true });
		}
	});
</script>

<div class="mx-auto flex min-h-dvh w-full max-w-md flex-col bg-base-100 text-base-content">
	<main class="flex-1 px-4 pt-4 pb-28">
		{#if !session.checked}
			<div class="flex min-h-[60dvh] items-center justify-center" aria-live="polite">
				<p class="text-sm opacity-70">Menyiapkan ANABUL...</p>
			</div>
		{:else}
			<!-- Halaman berganti dengan lembut supaya tidak terasa kaku. -->
			{#key jalur}
				<div in:fade={{ duration: durasiPindah }}>
					{@render children()}
				</div>
			{/key}
		{/if}
	</main>
</div>

{#if session.authenticated && !diLogin}
	<BottomNav active={jalur} />
{/if}

<Toasts />
