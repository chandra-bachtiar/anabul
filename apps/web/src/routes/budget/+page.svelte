<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { currentMonth, formatRupiah, shiftMonth, formatMonth, budgetStatusLabel } from '#lib/format';
	import { hapticLight } from '#lib/haptic';
	import { toast } from '#lib/toast.svelte';
	import type { BudgetRow, Category } from '#lib/types';

	let month = $state(currentMonth());
	let categories = $state<Category[]>([]);
	let budget = $state<BudgetRow[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let copying = $state(false);

	let selected = $state<Category | null>(null);
	let amount = $state(0);
	let saving = $state(false);

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			const result = await api.budgets(month);
			budget = result.budgets;
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat budget.';
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		try {
			categories = (await api.categories()).categories;
		} catch {
			toast('Gagal memuat daftar kategori.', 'error');
		}
		void load();
	});

	$effect(() => {
		month;
		void load();
	});

	const rows = $derived(
		categories.map((item) => {
			const match = budget.find((row) => row.category_id === item.id);
			return {
				category: item,
				row: match ?? {
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

	const totalBudget = $derived(budget.reduce((sum, row) => sum + row.amount, 0));
	const totalSpent = $derived(budget.reduce((sum, row) => sum + (row.amount > 0 ? row.spent : 0), 0));
	const remaining = $derived(totalBudget - totalSpent);

	function open(row: BudgetRow) {
		const category = categories.find((item) => item.id === row.category_id);
		if (!category) return;
		selected = category;
		amount = row.amount;
	}

	async function save() {
		if (!selected) return;
		saving = true;
		try {
			await api.saveBudgets(month, [{ category_id: selected.id, amount }]);
			selected = null;
			hapticLight();
			toast(amount === 0 ? 'Budget kategori itu dikosongkan.' : 'Budget tersimpan.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan budget.', 'error');
		} finally {
			saving = false;
		}
	}

	async function copyFromPrevious() {
		const previousMonth = shiftMonth(month, -1);
		copying = true;
		try {
			const result = await api.copyBudgets(previousMonth, month);
			toast(result.copied > 0 ? `${result.copied} budget disalin dari ${formatMonth(previousMonth)}.` : 'Tidak ada budget di bulan sebelumnya.', result.copied > 0 ? 'ok' : 'info');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyalin budget.', 'error');
		} finally {
			copying = false;
		}
	}
</script>

<svelte:head>
	<title>Budget — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Budget</h1>
		<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={copyFromPrevious} disabled={copying}>
			{copying ? 'Menyalin...' : `Salin dari ${formatMonth(shiftMonth(month, -1))}`}
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{formatMonth(month)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else}
		{#if totalBudget === 0}
			<div class="card border border-info/40 bg-info/10">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="happy" size={48} alt="" />
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
						status={remaining < 0 ? 'over' : totalSpent * 5 >= totalBudget * 4 ? 'warn' : 'ok'}
						label="Pemakaian total budget"
						thick
					/>
					<p class="text-sm">{budgetStatusLabel(remaining < 0 ? 'over' : 'ok')}, sisa {formatRupiah(remaining)}.</p>
				</div>
			</section>
		{/if}

		<ul class="space-y-2">
			{#each rows as item (item.category.id)}
				<li class="card bg-base-200">
					<button
						type="button"
						class="w-full rounded-box text-left transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
						onclick={() => open({ ...item.row, category_id: item.category.id, category_name: item.category.name })}
					>
						<div class="card-body gap-2 p-4">
							<div class="flex items-baseline justify-between gap-3">
								<span class="font-medium">{item.category.name}</span>
								<span class="font-display text-base font-semibold tabular-nums">
									{item.row.amount > 0 ? formatRupiah(item.row.remaining) : 'Belum diatur'}
								</span>
							</div>
							{#if item.row.amount > 0}
								<Progress value={item.row.progress} status={item.row.status} label={`Pemakaian ${item.category.name}`} />
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
	open={selected !== null}
	title={selected ? `Budget ${selected.name}` : 'Atur budget'}
	description="Isi 0 kalau kategori ini tidak perlu budget bulanan."
	onclose={() => (selected = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void save();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nominal-budget">Nominal budget</label>
			<MoneyInput id="nominal-budget" value={amount} onchange={(value) => (amount = value)} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (selected = null)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={saving}>
				{saving ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
