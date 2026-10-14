<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		title: string;
		description?: string;
		onclose: () => void;
		children: Snippet;
		footer?: Snippet;
	}

	let { open, title, description, onclose, children, footer }: Props = $props();

	let dialog = $state<HTMLDialogElement | null>(null);

	// Setiap sheet butuh id unik karena satu halaman bisa punya beberapa sheet.
	const idJudul = `judul-sheet-${Math.random().toString(36).slice(2, 8)}`;

	// Dialog bawaan browser sudah menangani Escape, fokus, dan backdrop.
	$effect(() => {
		const node = dialog;
		if (!node) return;
		if (open && !node.open) {
			node.showModal();
		} else if (!open && node.open) {
			node.close();
		}
	});
</script>

<dialog
	bind:this={dialog}
	class="modal modal-bottom sm:modal-middle"
	aria-labelledby={idJudul}
	onclose={onclose}
>
	<div class="modal-box max-h-[88dvh] w-full max-w-md rounded-t-3xl sm:rounded-3xl">
		<div class="flex items-start justify-between gap-3">
			<div>
				<h2 id={idJudul} class="font-display text-lg font-semibold">{title}</h2>
				{#if description}
					<p class="mt-1 text-sm opacity-70">{description}</p>
				{/if}
			</div>
			<button
				type="button"
				class="btn btn-circle btn-ghost min-h-11 min-w-11"
				onclick={onclose}
				aria-label="Tutup panel"
			>
				<svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" aria-hidden="true">
					<path
						d="M7 7l10 10M17 7L7 17"
						stroke="currentColor"
						stroke-width="1.8"
						stroke-linecap="round"
					/>
				</svg>
			</button>
		</div>

		<div class="mt-4">
			{@render children()}
		</div>

		{#if footer}
			<div class="mt-5 flex flex-wrap justify-end gap-2">
				{@render footer()}
			</div>
		{/if}
	</div>
	<form method="dialog" class="modal-backdrop">
		<button type="button" aria-label="Tutup panel">tutup</button>
	</form>
</dialog>
