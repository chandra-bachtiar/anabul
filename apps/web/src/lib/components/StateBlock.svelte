<script lang="ts">
	import Mascot from './Mascot.svelte';

	interface Props {
		state: 'loading' | 'empty' | 'error';
		title?: string;
		message?: string;
		actionLabel?: string;
		onaction?: () => void;
	}

	let { state, title, message, actionLabel, onaction }: Props = $props();
</script>

{#if state === 'loading'}
	<div class="space-y-3" aria-busy="true" aria-live="polite">
		<span class="sr-only">Sedang memuat data</span>
		<div class="card bg-base-200">
			<div class="card-body gap-3">
				<div class="skeleton h-5 w-1/3"></div>
				<div class="skeleton h-8 w-2/3"></div>
			</div>
		</div>
		<div class="card bg-base-200">
			<div class="card-body gap-3">
				<div class="skeleton h-4 w-1/2"></div>
				<div class="skeleton h-4 w-3/4"></div>
				<div class="skeleton h-4 w-2/5"></div>
			</div>
		</div>
	</div>
{:else if state === 'empty'}
	<div class="card bg-base-200">
		<div class="card-body items-center gap-2 py-8 text-center">
			<Mascot mood="asleep" size={104} alt="" />
			<p class="font-medium">{title ?? 'Belum ada catatan'}</p>
			{#if message}
				<p class="max-w-[28ch] text-sm opacity-70">{message}</p>
			{/if}
			{#if actionLabel && onaction}
				<button type="button" class="btn btn-primary mt-2 min-h-11" onclick={onaction}>
					{actionLabel}
				</button>
			{/if}
		</div>
	</div>
{:else}
	<div class="card border border-error/40 bg-base-200">
		<div class="card-body gap-3">
			<p class="font-medium text-error-ink">{title ?? 'Gagal memuat data'}</p>
			<p class="text-sm">{message ?? 'Ada gangguan saat menghubungi server.'}</p>
			{#if actionLabel && onaction}
				<button type="button" class="btn btn-outline min-h-11 self-start" onclick={onaction}>
					{actionLabel}
				</button>
			{/if}
		</div>
	</div>
{/if}
