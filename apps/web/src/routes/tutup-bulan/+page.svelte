<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Confetti from '#lib/components/Confetti.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { currentMonth, formatRupiah, shiftMonth, formatMonth, formatDateShort } from '#lib/format';
	import { hapticCelebrate } from '#lib/haptic';
	import { toast } from '#lib/toast.svelte';
	import type { MonthPreview, Snapshot } from '#lib/types';

	let month = $state(currentMonth());
	let preview = $state<MonthPreview | null>(null);
	let loading = $state(true);
	let errorMessage = $state('');
	let processing = $state(false);
	let confirmOpen = $state(false);
	let result = $state<Snapshot | null>(null);
	let celebration = $state(0);

	async function load() {
		loading = true;
		errorMessage = '';
		result = null;
		try {
			preview = await api.monthPreview(month);
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat ringkasan bulan.';
		} finally {
			loading = false;
		}
	}

	onMount(() => void load());

	$effect(() => {
		month;
		void load();
	});

	async function closeMonth() {
		processing = true;
		try {
			result = await api.closeMonth(month);
			confirmOpen = false;
			celebration += 1;
			hapticCelebrate();
			toast(`Bulan ${formatMonth(month)} sudah ditutup.`);
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menutup bulan.', 'error');
		} finally {
			processing = false;
		}
	}

	async function reopenMonth() {
		processing = true;
		try {
			await api.reopenMonth(month);
			toast(`Bulan ${formatMonth(month)} bisa diubah lagi.`);
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal membuka bulan.', 'error');
		} finally {
			processing = false;
		}
	}
</script>

<svelte:head>
	<title>Tutup bulan — ANABUL</title>
</svelte:head>

<Confetti trigger={celebration} />

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Tutup bulan</h1>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{formatMonth(month)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else if preview}
		{#if result}
			<section class="card border border-success/40 bg-success/10" aria-labelledby="judul-hasil">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="happy" size={56} alt="" />
					<div>
						<h2 id="judul-hasil" class="font-display text-base font-semibold">Bulan {formatMonth(result.month)} terkunci</h2>
						<p class="text-sm">
							Pemasukan {formatRupiah(result.income)}, pengeluaran {formatRupiah(result.expense)}, selisih
							{formatRupiah(result.net)}. Bulan berikutnya sudah disiapkan dengan {result.carried_budgets} budget
							dan {result.carried_needs} kebutuhan.
						</p>
					</div>
				</div>
			</section>
		{/if}

		{#if preview.locked}
			<section class="card border border-warning/50 bg-warning/10" aria-labelledby="judul-terkunci">
				<div class="card-body flex-row items-start gap-3 p-4">
					<Mascot mood="asleep" size={56} alt="" />
					<div class="space-y-2">
						<h2 id="judul-terkunci" class="font-display text-base font-semibold">Bulan ini terkunci</h2>
						<p class="text-sm">
							Transaksi dan budget bulan ini tidak bisa diubah. Buka kembali hanya kalau memang ada yang
							perlu dikoreksi.
						</p>
						<button type="button" class="btn btn-outline min-h-11" onclick={reopenMonth} disabled={processing}>
							{processing ? 'Membuka...' : 'Buka kembali bulan ini'}
						</button>
					</div>
				</div>
			</section>
		{/if}

		<section class="card bg-base-200" aria-labelledby="judul-ringkasan">
			<div class="card-body gap-3 p-4">
				<h2 id="judul-ringkasan" class="font-display text-base font-semibold">Ringkasan {preview.month_label}</h2>
				<dl class="grid grid-cols-3 gap-2 text-sm">
					<div>
						<dt class="text-xs opacity-70">Masuk</dt>
						<dd class="font-display text-lg font-semibold tabular-nums text-success-ink">{formatRupiah(preview.income)}</dd>
					</div>
					<div class="text-center">
						<dt class="text-xs opacity-70">Keluar</dt>
						<dd class="font-display text-lg font-semibold tabular-nums text-error-ink">{formatRupiah(preview.expense)}</dd>
					</div>
					<div class="text-right">
						<dt class="text-xs opacity-70">Selisih</dt>
						<dd class="font-display text-lg font-semibold tabular-nums">{formatRupiah(preview.net)}</dd>
					</div>
				</dl>
				<p class="text-xs opacity-70">{preview.categories.length} kategori terpakai, {preview.budgets.length} budget diatur.</p>
			</div>
		</section>

		{#if preview.budgets.length > 0}
			<section class="card bg-base-200" aria-labelledby="close-budget-title">
				<div class="card-body gap-3 p-4">
					<h2 id="close-budget-title" class="font-display text-base font-semibold">Sisa budget per kategori</h2>
					<ul class="space-y-3">
						{#each preview.budgets as item (item.category_id)}
							<li class="space-y-1">
								<div class="flex items-baseline justify-between gap-3 text-sm">
									<span class="font-medium">{item.category_name}</span>
									<span class="tabular-nums {item.remaining < 0 ? 'font-semibold text-error-ink' : 'opacity-80'}">
										{item.remaining < 0 ? 'Lewat ' : 'Sisa '}{formatRupiah(item.remaining < 0 ? -item.remaining : item.remaining)}
									</span>
								</div>
								<Progress value={item.progress} status={item.status} label={`Pemakaian ${item.category_name}`} />
							</li>
						{/each}
					</ul>
					<p class="text-xs opacity-70">Aturan sisa budget: hangus saat tutup bulan, tidak dibawa ke bulan berikutnya.</p>
				</div>
			</section>
		{/if}

		{#if preview.unpaid_need_rows.length > 0}
			<section class="card bg-base-200" aria-labelledby="judul-belum-bayar">
				<div class="card-body gap-2 p-4">
					<h2 id="judul-belum-bayar" class="font-display text-base font-semibold">Kebutuhan belum dibayar</h2>
					<ul class="space-y-1 text-sm">
						{#each preview.unpaid_need_rows as item (item.id)}
							<li class="flex items-baseline justify-between gap-3">
								<span>{item.name} · {formatDateShort(item.due_date)}</span>
								<span class="tabular-nums">{formatRupiah(item.amount)}</span>
							</li>
						{/each}
					</ul>
					<p class="text-xs opacity-70">Kebutuhan ini tetap terbuka di bulan berikutnya.</p>
				</div>
			</section>
		{/if}

		{#if preview.due_debts.length > 0 || preview.debts.hutang > 0 || preview.debts.piutang > 0}
			<section class="card bg-base-200" aria-labelledby="judul-hutang-tutup">
				<div class="card-body gap-2 p-4">
					<h2 id="judul-hutang-tutup" class="font-display text-base font-semibold">Hutang piutang berjalan</h2>
					<p class="text-sm">
						Hutang {formatRupiah(preview.debts.hutang)} · piutang {formatRupiah(preview.debts.piutang)}
					</p>
					{#if preview.due_debts.length > 0}
						<ul class="space-y-1 text-sm">
							{#each preview.due_debts as item (item.id)}
								<li class="flex items-baseline justify-between gap-3">
									<span>
										{item.direction === 'hutang' ? 'Bayar ke' : 'Tagih'} {item.person}
										{#if item.due_date}· {formatDateShort(item.due_date)}{/if}
									</span>
									<span class="tabular-nums">{formatRupiah(item.remaining)}</span>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</section>
		{/if}

		{#if preview.warnings.length > 0}
			<div class="card border border-info/40 bg-info/10">
				<div class="card-body gap-2 p-4">
					<h2 class="font-display text-base font-semibold">Sebelum ditutup, cek ini</h2>
					<ul class="list-disc space-y-1 pl-5 text-sm">
						{#each preview.warnings as warning, index (index)}
							<li>{warning}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}

		{#if !preview.locked}
			<button type="button" class="btn btn-primary min-h-12 w-full text-base" onclick={() => (confirmOpen = true)}>
				Tutup bulan {formatMonth(month)}
			</button>
		{/if}
	{/if}
</div>

<Sheet
	open={confirmOpen}
	title={`Tutup ${formatMonth(month)}?`}
	description="Setelah ditutup, transaksi bulan ini tidak bisa diubah lagi kecuali kamu buka kembali."
	onclose={() => (confirmOpen = false)}
>
	<div class="space-y-4">
		<ul class="list-disc space-y-1 pl-5 text-sm">
			<li>Ringkasan bulan ini disimpan sebagai snapshot.</li>
			<li>Budget bulan ini disalin ke bulan berikutnya.</li>
			<li>Kebutuhan bulanan berikutnya disiapkan dari template.</li>
			<li>Sisa budget bulan ini hangus, tidak dibawa ke bulan depan.</li>
		</ul>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (confirmOpen = false)}>Batal</button>
			<button type="button" class="btn btn-primary min-h-11" onclick={closeMonth} disabled={processing}>
				{processing ? 'Menutup...' : 'Ya, tutup sekarang'}
			</button>
		</div>
	</div>
</Sheet>
