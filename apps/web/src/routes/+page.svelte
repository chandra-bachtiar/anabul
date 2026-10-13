<script lang="ts">
	import { onMount } from 'svelte';

	let serverStatus = $state<'checking' | 'ok' | 'down'>('checking');

	onMount(async () => {
		try {
			const res = await fetch('/healthz');
			serverStatus = res.ok ? 'ok' : 'down';
		} catch {
			serverStatus = 'down';
		}
	});
</script>

<svelte:head>
	<title>ANABUL — Asisten Ngatur Anggaran Bulanan</title>
</svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 p-4 pb-8">
	<header class="flex items-center gap-3 pt-4">
		<img src="/mascot-senang.svg" alt="Maskot ANABUL" class="h-16 w-16" />
		<div>
			<h1 class="font-display text-3xl font-semibold tracking-wide text-primary">ANABUL</h1>
			<p class="text-sm opacity-70">Asisten Ngatur Anggaran Bulanan</p>
		</div>
	</header>

	<section class="card bg-base-200 shadow-sm">
		<div class="card-body gap-2">
			<h2 class="card-title text-base">Status</h2>
			{#if serverStatus === 'checking'}
				<span class="badge badge-ghost">Mengecek server…</span>
			{:else if serverStatus === 'ok'}
				<span class="badge badge-success">Server dan database siap</span>
			{:else}
				<span class="badge badge-error">Server tidak merespons</span>
				<p class="text-sm opacity-70">
					Jalankan <code class="rounded bg-base-300 px-1">make dev-api</code> lalu muat ulang
					halaman ini.
				</p>
			{/if}
		</div>
	</section>

	<section class="card bg-base-200 shadow-sm">
		<div class="card-body items-center gap-2 text-center">
			<img src="/mascot-tidur.svg" alt="" class="h-28 w-28" />
			<p class="font-medium">Belum ada catatan.</p>
			<p class="text-sm opacity-70">
				Kerangka aplikasi sudah jalan. Input chat seperti "kopi 25rb" menyusul di fase
				berikutnya, meong.
			</p>
		</div>
	</section>
</main>
