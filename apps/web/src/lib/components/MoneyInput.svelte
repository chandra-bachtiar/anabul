<script lang="ts">
	interface Props {
		value: number;
		id: string;
		onchange: (value: number) => void;
		placeholder?: string;
		required?: boolean;
	}

	let { value, id, onchange, placeholder = '0', required = false }: Props = $props();

	const numberFormat = new Intl.NumberFormat('id-ID');
	const display = $derived(value > 0 ? numberFormat.format(value) : '');
</script>

<!-- Amounts never have cents, so typing shows the thousands separators and
     every other character is dropped. -->
<input
	{id}
	type="text"
	inputmode="numeric"
	{required}
	value={display}
	{placeholder}
	class="input input-bordered min-h-11 w-full text-right font-display text-xl tabular-nums"
	oninput={(event) => {
		const digits = event.currentTarget.value.replace(/\D/g, '');
		onchange(digits ? Number(digits) : 0);
	}}
/>
