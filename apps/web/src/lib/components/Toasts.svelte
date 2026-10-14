<script lang="ts">
	import { dismiss, toasts } from '#lib/toast.svelte';
</script>

<div
	class="pointer-events-none fixed inset-x-0 bottom-24 z-40 flex flex-col items-center gap-2 px-4"
	aria-live="polite"
	aria-atomic="false"
>
	{#each toasts.items as item (item.id)}
		<button
			type="button"
			class="pointer-events-auto flex min-h-11 w-full max-w-sm items-center gap-3 rounded-2xl px-4 py-2 text-left text-sm font-medium shadow-lg {item.tone ===
			'error'
				? 'bg-error text-error-ink-content'
				: item.tone === 'info'
					? 'bg-accent text-accent-content'
					: 'bg-success text-success-ink-content'}"
			onclick={() => dismiss(item.id)}
		>
			<span class="flex-1">{item.message}</span>
			<span class="text-xs opacity-80">tutup</span>
		</button>
	{/each}
</div>
