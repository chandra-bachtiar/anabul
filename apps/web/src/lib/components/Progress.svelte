<script lang="ts">
	interface Props {
		value: number;
		status?: 'ok' | 'warn' | 'over' | 'none';
		label: string;
		thick?: boolean;
	}

	let { value, status = 'ok', label, thick = false }: Props = $props();

	const clamped = $derived(Math.max(0, Math.min(value, 100)));

	const color = $derived(
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
	class="w-full overflow-hidden rounded-full bg-base-300 {thick ? 'h-3' : 'h-2'}"
	role="progressbar"
	aria-label={label}
	aria-valuenow={Math.round(value)}
	aria-valuemin={0}
	aria-valuemax={100}
>
	<div
		class="h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none {color}"
		style="width: {clamped}%"
	></div>
</div>
