<script lang="ts">
	interface Props {
		value: number;
		status?: 'ok' | 'warn' | 'over' | 'none';
		label: string;
		tebal?: boolean;
	}

	let { value, status = 'ok', label, tebal = false }: Props = $props();

	const terlihat = $derived(Math.max(0, Math.min(value, 100)));

	const warna = $derived(
		status === 'over'
			? 'bg-error'
			: status === 'warn'
				? 'bg-warning'
				: status === 'none'
					? 'bg-base-content/25'
					: 'bg-success'
	);
</script>

<div
	class="w-full overflow-hidden rounded-full bg-base-300 {tebal ? 'h-3' : 'h-2'}"
	role="progressbar"
	aria-label={label}
	aria-valuenow={Math.round(value)}
	aria-valuemin={0}
	aria-valuemax={100}
>
	<div
		class="h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none {warna}"
		style="width: {terlihat}%"
	></div>
</div>
