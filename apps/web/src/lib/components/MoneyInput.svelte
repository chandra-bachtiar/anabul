<script lang="ts">
	interface Props {
		value: number;
		id: string;
		onchange: (value: number) => void;
		placeholder?: string;
		required?: boolean;
	}

	let { value, id, onchange, placeholder = '0', required = false }: Props = $props();

	const angka = new Intl.NumberFormat('id-ID');
	const tampil = $derived(value > 0 ? angka.format(value) : '');
</script>

<!-- Nominal ditulis sebagai rupiah bulat, jadi pemisah ribuan langsung muncul
     dan huruf lain diabaikan. -->
<input
	{id}
	type="text"
	inputmode="numeric"
	{required}
	value={tampil}
	{placeholder}
	class="input input-bordered min-h-11 w-full text-right font-display text-xl tabular-nums"
	oninput={(event) => {
		const digit = event.currentTarget.value.replace(/\D/g, '');
		onchange(digit ? Number(digit) : 0);
	}}
/>
