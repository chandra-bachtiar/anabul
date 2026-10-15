<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import BottomNav from '#lib/components/BottomNav.svelte';
	import Toasts from '#lib/components/Toasts.svelte';
	import { motionAllowed } from '#lib/haptic';
	import { checkSession, session } from '#lib/session.svelte';
	import { initTheme } from '#lib/theme.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const pathname = $derived(page.url.pathname);
	const onLoginPage = $derived(pathname === '/login');
	const transitionDuration = $derived(motionAllowed() ? 180 : 0);

	onMount(() => {
		initTheme();
		void checkSession();

		// Service worker is production-only; during dev it interferes with HMR.
		if (import.meta.env.PROD && 'serviceWorker' in navigator) {
			navigator.serviceWorker.register('/sw.js').catch(() => {
				// Offline support is optional, the app still runs online.
			});
		}
	});

	$effect(() => {
		if (session.checked && !session.authenticated && !onLoginPage) {
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
			{#key pathname}
				<div in:fade={{ duration: transitionDuration }}>
					{@render children()}
				</div>
			{/key}
		{/if}
	</main>
</div>

{#if session.authenticated && !onLoginPage}
	<BottomNav active={pathname} />
{/if}

<Toasts />
