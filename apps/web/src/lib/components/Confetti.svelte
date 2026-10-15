<script lang="ts">
	import { motionAllowed } from '#lib/haptic';

	interface Props {
		/** Bumped every time a celebration is requested. */
		trigger?: number;
	}

	let { trigger = 0 }: Props = $props();

	let isVisible = $state(false);

	// Confetti renders for a moment only; reduced-motion users see nothing,
	// which is the point of prefers-reduced-motion.
	$effect(() => {
		if (trigger === 0 || !motionAllowed()) return;
		isVisible = true;
		const timer = setTimeout(() => (isVisible = false), 1500);
		return () => clearTimeout(timer);
	});

	const colors = ['#ff9f43', '#ffb45e', '#5bb5f0', '#3cc48b', '#ff8fa3', '#ffc145'];

	function styleFor(index: number): string {
		const left = (index * 37) % 100;
		const delay = (index % 7) * 40;
		const fall = 120 + ((index * 29) % 90);
		const spin = 240 + ((index * 41) % 240);
		return `left:${left}%;animation-delay:${delay}ms;background:${colors[index % colors.length]};--fall:${fall}px;--spin:${spin}deg`;
	}
</script>

{#if isVisible}
	<div class="pointer-events-none fixed inset-x-0 top-20 z-50 mx-auto h-0 max-w-md" aria-hidden="true">
		{#each Array(24) as _, index (index)}
			<span class="anabul-confetti absolute block h-2 w-2 rounded-sm" style={styleFor(index)}></span>
		{/each}
	</div>
{/if}
