<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Mascot from '#lib/components/Mascot.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { bulanIni, formatRupiah, geserBulan, labelBulan, labelTanggalPendek } from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { MonthPreview, Snapshot } from '#lib/types';

	let bulan = $state(bulanIni());
	let pratinjau = $state<MonthPreview | null>(null);
	let memuat = $state(true);
	let pesanError = $state('');
	let memproses = $state(false);
	let konfirmasiTerbuka = $state(false);
	let hasil = $state<Snapshot | null>(null);

	async function muat() {
		memuat = true;
		pesanError = '';
		hasil = null;
		try {
			pratinjau = await api.monthPreview(bulan);
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat ringkasan bulan.';
		} finally {
			memuat = false;
		}
	}

	onMount(() => void muat());

	$effect(() => {
		bulan;
		void muat();
	});

	async function tutupBulan() {
		memproses = true;
		try {
			hasil = await api.closeMonth(bulan);
			konfirmasiTerbuka = false;
			toast(`Bulan ${labelBulan(bulan)} sudah ditutup.`);
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menutup bulan.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function bukaKembali() {
		memproses = true;
		try {
			await api.reopenMonth(bulan);
			toast(`Bulan ${labelBulan(bulan)} bisa diubah lagi.`);
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal membuka bulan.', 'error');
		} finally {
			memproses = false;
		}
	}
</script>

<svelte:head>
	<title>Tutup bulan — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Tutup bulan</h1>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{labelBulan(bulan)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else if pratinjau}
		{#if hasil}
			<section class="card border border-success/40 bg-success/10" aria-labelledby="judul-hasil">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="senang" size={56} alt="" />
					<div>
						<h2 id="judul-hasil" class="font-display text-base font-semibold">Bulan {labelBulan(hasil.month)} terkunci</h2>
						<p class="text-sm">
							Pemasukan {formatRupiah(hasil.income)}, pengeluaran {formatRupiah(hasil.expense)}, selisih
							{formatRupiah(hasil.net)}. Bulan berikutnya sudah disiapkan dengan {hasil.carried_budgets} budget
							dan {hasil.carried_needs} kebutuhan.
						</p>
					</div>
				</div>
			</section>
		{/if}

		{#if pratinjau.locked}
			<section class="card border border-warning/50 bg-warning/10" aria-labelledby="judul-terkunci">
				<div class="card-body flex-row items-start gap-3 p-4">
					<Mascot mood="tidur" size={56} alt="" />
					<div class="space-y-2">
						<h2 id="judul-terkunci" class="font-display text-base font-semibold">Bulan ini terkunci</h2>
						<p class="text-sm">
							Transaksi dan budget bulan ini tidak bisa diubah. Buka kembali hanya kalau memang ada yang
							perlu dikoreksi.
						</p>
						<button type="button" class="btn btn-outline min-h-11" onclick={bukaKembali} disabled={memproses}>
							{memproses ? 'Membuka...' : 'Buka kembali bulan ini'}
						</button>
					</div>
				</div>
			</section>
		{/if}

		<section class="card bg-base-200" aria-labelledby="judul-ringkasan">
			<div class="card-body gap-3 p-4">
				<h2 id="judul-ringkasan" class="font-display text-base font-semibold">Ringkasan {pratinjau.month_label}</h2>
				<dl class="grid grid-cols-3 gap-2 text-sm">
					<div>
						<dt class="text-xs opacity-70">Masuk</dt>
						<dd class="font-display text-lg font-semibold tabular-nums text-success-ink">{formatRupiah(pratinjau.income)}</dd>
					</div>
					<div class="text-center">
						<dt class="text-xs opacity-70">Keluar</dt>
						<dd class="font-display text-lg font-semibold tabular-nums text-error-ink">{formatRupiah(pratinjau.expense)}</dd>
					</div>
					<div class="text-right">
						<dt class="text-xs opacity-70">Selisih</dt>
						<dd class="font-display text-lg font-semibold tabular-nums">{formatRupiah(pratinjau.net)}</dd>
					</div>
				</dl>
				<p class="text-xs opacity-70">{pratinjau.categories.length} kategori terpakai, {pratinjau.budgets.length} budget diatur.</p>
			</div>
		</section>

		{#if pratinjau.budgets.length > 0}
			<section class="card bg-base-200" aria-labelledby="judul-budget-tutup">
				<div class="card-body gap-3 p-4">
					<h2 id="judul-budget-tutup" class="font-display text-base font-semibold">Sisa budget per kategori</h2>
					<ul class="space-y-3">
						{#each pratinjau.budgets as item (item.category_id)}
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

		{#if pratinjau.unpaid_need_rows.length > 0}
			<section class="card bg-base-200" aria-labelledby="judul-belum-bayar">
				<div class="card-body gap-2 p-4">
					<h2 id="judul-belum-bayar" class="font-display text-base font-semibold">Kebutuhan belum dibayar</h2>
					<ul class="space-y-1 text-sm">
						{#each pratinjau.unpaid_need_rows as item (item.id)}
							<li class="flex items-baseline justify-between gap-3">
								<span>{item.name} · {labelTanggalPendek(item.due_date)}</span>
								<span class="tabular-nums">{formatRupiah(item.amount)}</span>
							</li>
						{/each}
					</ul>
					<p class="text-xs opacity-70">Kebutuhan ini tetap terbuka di bulan berikutnya.</p>
				</div>
			</section>
		{/if}

		{#if pratinjau.due_debts.length > 0 || pratinjau.debts.hutang > 0 || pratinjau.debts.piutang > 0}
			<section class="card bg-base-200" aria-labelledby="judul-hutang-tutup">
				<div class="card-body gap-2 p-4">
					<h2 id="judul-hutang-tutup" class="font-display text-base font-semibold">Hutang piutang berjalan</h2>
					<p class="text-sm">
						Hutang {formatRupiah(pratinjau.debts.hutang)} · piutang {formatRupiah(pratinjau.debts.piutang)}
					</p>
					{#if pratinjau.due_debts.length > 0}
						<ul class="space-y-1 text-sm">
							{#each pratinjau.due_debts as item (item.id)}
								<li class="flex items-baseline justify-between gap-3">
									<span>
										{item.direction === 'hutang' ? 'Bayar ke' : 'Tagih'} {item.person}
										{#if item.due_date}· {labelTanggalPendek(item.due_date)}{/if}
									</span>
									<span class="tabular-nums">{formatRupiah(item.remaining)}</span>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</section>
		{/if}

		{#if pratinjau.warnings.length > 0}
			<div class="card border border-info/40 bg-info/10">
				<div class="card-body gap-2 p-4">
					<h2 class="font-display text-base font-semibold">Sebelum ditutup, cek ini</h2>
					<ul class="list-disc space-y-1 pl-5 text-sm">
						{#each pratinjau.warnings as pesan, index (index)}
							<li>{pesan}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}

		{#if !pratinjau.locked}
			<button type="button" class="btn btn-primary min-h-12 w-full text-base" onclick={() => (konfirmasiTerbuka = true)}>
				Tutup bulan {labelBulan(bulan)}
			</button>
		{/if}
	{/if}
</div>

<Sheet
	open={konfirmasiTerbuka}
	title={`Tutup ${labelBulan(bulan)}?`}
	description="Setelah ditutup, transaksi bulan ini tidak bisa diubah lagi kecuali kamu buka kembali."
	onclose={() => (konfirmasiTerbuka = false)}
>
	<div class="space-y-4">
		<ul class="list-disc space-y-1 pl-5 text-sm">
			<li>Ringkasan bulan ini disimpan sebagai snapshot.</li>
			<li>Budget bulan ini disalin ke bulan berikutnya.</li>
			<li>Kebutuhan bulanan berikutnya disiapkan dari template.</li>
			<li>Sisa budget bulan ini hangus, tidak dibawa ke bulan depan.</li>
		</ul>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (konfirmasiTerbuka = false)}>Batal</button>
			<button type="button" class="btn btn-primary min-h-11" onclick={tutupBulan} disabled={memproses}>
				{memproses ? 'Menutup...' : 'Ya, tutup sekarang'}
			</button>
		</div>
	</div>
</Sheet>
