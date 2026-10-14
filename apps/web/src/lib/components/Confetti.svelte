<script lang="ts">
	import { bolehBergerak } from '#lib/haptic';

	interface Props {
		/** Angka ini berubah setiap kali perayaan diminta. */
		trigger?: number;
	}

	let { trigger = 0 }: Props = $props();

	let aktif = $state(false);

	// Konfeti hanya dirender sesaat. Pengguna yang meminta kurangi gerak tidak
	// melihat apa pun, jadi konsisten dengan prefers-reduced-motion.
	$effect(() => {
		if (trigger === 0 || !bolehBergerak()) return;
		aktif = true;
		const timer = setTimeout(() => (aktif = false), 1500);
		return () => clearTimeout(timer);
	});

	const warna = ['#ff9f43', '#ffb45e', '#5bb5f0', '#3cc48b', '#ff8fa3', '#ffc145'];

	function gaya(index: number): string {
		const kiri = (index * 37) % 100;
		const tunda = (index % 7) * 40;
		const jatuh = 120 + ((index * 29) % 90);
		const putar = 240 + ((index * 41) % 240);
		return `left:${kiri}%;animation-delay:${tunda}ms;background:${warna[index % warna.length]};--jatuh:${jatuh}px;--putar:${putar}deg`;
	}
</script>

{#if aktif}
	<div class="pointer-events-none fixed inset-x-0 top-20 z-50 mx-auto h-0 max-w-md" aria-hidden="true">
		{#each Array(24) as _, index (index)}
			<span class="anabul-konfeti absolute block h-2 w-2 rounded-sm" style={gaya(index)}></span>
		{/each}
	</div>
{/if}
