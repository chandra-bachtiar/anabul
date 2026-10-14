<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { bulanIni, formatRupiah, geserBulan, labelBulan, statusBudgetLabel } from '#lib/format';
	import { getarRingan } from '#lib/haptic';
	import { toast } from '#lib/toast.svelte';
	import type { BudgetRow, Category } from '#lib/types';

	let bulan = $state(bulanIni());
	let kategori = $state<Category[]>([]);
	let budget = $state<BudgetRow[]>([]);
	let memuat = $state(true);
	let pesanError = $state('');
	let menyalin = $state(false);

	let dipilih = $state<Category | null>(null);
	let nominal = $state(0);
	let menyimpan = $state(false);

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			const hasil = await api.budgets(bulan);
			budget = hasil.budgets;
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat budget.';
		} finally {
			memuat = false;
		}
	}

	onMount(async () => {
		try {
			kategori = (await api.categories()).categories;
		} catch {
			toast('Gagal memuat daftar kategori.', 'error');
		}
		void muat();
	});

	$effect(() => {
		bulan;
		void muat();
	});

	const baris = $derived(
		kategori.map((item) => {
			const cocok = budget.find((row) => row.category_id === item.id);
			return {
				kategori: item,
				row: cocok ?? {
					category_id: item.id,
					category_name: item.name,
					amount: 0,
					spent: 0,
					remaining: 0,
					status: 'none' as const,
					progress: 0
				}
			};
		})
	);

	const totalBudget = $derived(budget.reduce((jumlah, row) => jumlah + row.amount, 0));
	const totalSpent = $derived(budget.reduce((jumlah, row) => jumlah + (row.amount > 0 ? row.spent : 0), 0));
	const sisa = $derived(totalBudget - totalSpent);

	function buka(row: BudgetRow) {
		const item = kategori.find((k) => k.id === row.category_id);
		if (!item) return;
		dipilih = item;
		nominal = row.amount;
	}

	async function simpan() {
		if (!dipilih) return;
		menyimpan = true;
		try {
			await api.saveBudgets(bulan, [{ category_id: dipilih.id, amount: nominal }]);
			dipilih = null;
			getarRingan();
			toast(nominal === 0 ? 'Budget kategori itu dikosongkan.' : 'Budget tersimpan.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan budget.', 'error');
		} finally {
			menyimpan = false;
		}
	}

	async function salin() {
		const sebelumnya = geserBulan(bulan, -1);
		menyalin = true;
		try {
			const hasil = await api.copyBudgets(sebelumnya, bulan);
			toast(hasil.copied > 0 ? `${hasil.copied} budget disalin dari ${labelBulan(sebelumnya)}.` : 'Tidak ada budget di bulan sebelumnya.', hasil.copied > 0 ? 'ok' : 'info');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyalin budget.', 'error');
		} finally {
			menyalin = false;
		}
	}
</script>

<svelte:head>
	<title>Budget — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Budget</h1>
		<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={salin} disabled={menyalin}>
			{menyalin ? 'Menyalin...' : `Salin dari ${labelBulan(geserBulan(bulan, -1))}`}
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{labelBulan(bulan)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else}
		{#if totalBudget === 0}
			<div class="card border border-info/40 bg-info/10">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="senang" size={48} alt="" />
					<p class="text-sm">
						Belum ada budget bulan ini. Ketuk kategori di bawah untuk mengisi nominalnya, atau salin
						dari bulan sebelumnya.
					</p>
				</div>
			</div>
		{:else}
			<section class="card bg-base-200" aria-labelledby="judul-total">
				<div class="card-body gap-3 p-4">
					<h2 id="judul-total" class="text-sm font-semibold opacity-70">Total budget terpakai</h2>
					<p class="font-display text-3xl leading-tight font-semibold tabular-nums">
						{formatRupiah(totalSpent)}
						<span class="text-base font-medium opacity-70">dari {formatRupiah(totalBudget)}</span>
					</p>
					<Progress
						value={totalBudget > 0 ? (totalSpent * 100) / totalBudget : 0}
						status={sisa < 0 ? 'over' : totalSpent * 5 >= totalBudget * 4 ? 'warn' : 'ok'}
						label="Pemakaian total budget"
						tebal
					/>
					<p class="text-sm">{statusBudgetLabel(sisa < 0 ? 'over' : 'ok')}, sisa {formatRupiah(sisa)}.</p>
				</div>
			</section>
		{/if}

		<ul class="space-y-2">
			{#each baris as item (item.kategori.id)}
				<li class="card bg-base-200">
					<button
						type="button"
						class="w-full rounded-box text-left transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
						onclick={() => buka({ ...item.row, category_id: item.kategori.id, category_name: item.kategori.name })}
					>
						<div class="card-body gap-2 p-4">
							<div class="flex items-baseline justify-between gap-3">
								<span class="font-medium">{item.kategori.name}</span>
								<span class="font-display text-base font-semibold tabular-nums">
									{item.row.amount > 0 ? formatRupiah(item.row.remaining) : 'Belum diatur'}
								</span>
							</div>
							{#if item.row.amount > 0}
								<Progress value={item.row.progress} status={item.row.status} label={`Pemakaian ${item.kategori.name}`} />
								<p class="text-xs opacity-70">
									Terpakai {formatRupiah(item.row.spent)} dari {formatRupiah(item.row.amount)}
								</p>
							{:else if item.row.spent > 0}
								<p class="text-xs opacity-70">
									Sudah keluar {formatRupiah(item.row.spent)} tanpa budget. Ketuk untuk mengisinya.
								</p>
							{:else}
								<p class="text-xs opacity-70">Ketuk untuk mengisi budget kategori ini.</p>
							{/if}
						</div>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Sheet
	open={dipilih !== null}
	title={dipilih ? `Budget ${dipilih.name}` : 'Atur budget'}
	description="Isi 0 kalau kategori ini tidak perlu budget bulanan."
	onclose={() => (dipilih = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void simpan();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nominal-budget">Nominal budget</label>
			<MoneyInput id="nominal-budget" value={nominal} onchange={(value) => (nominal = value)} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (dipilih = null)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={menyimpan}>
				{menyimpan ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
