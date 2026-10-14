<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { formatRupiah, labelTanggal, labelTanggalPendek } from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Debt, DebtTotals } from '#lib/types';

	let arah = $state<'hutang' | 'piutang'>('hutang');
	let daftar = $state<Debt[]>([]);
	let total = $state<DebtTotals>({ hutang: 0, piutang: 0 });
	let memuat = $state(true);
	let pesanError = $state('');
	let tampilLunas = $state(false);

	let detail = $state<Debt | null>(null);
	let nominalBayar = $state(0);
	let memproses = $state(false);

	let tambahTerbuka = $state(false);
	let formBaru = $state({ person: '', amount: 0, note: '', due_date: '' });

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			const hasil = await api.debts(tampilLunas ? undefined : 'open');
			daftar = hasil.debts;
			total = hasil.totals;
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat hutang piutang.';
		} finally {
			memuat = false;
		}
	}

	onMount(() => void muat());

	const terlihat = $derived(daftar.filter((item) => item.direction === arah));

	async function bukaDetail(id: number) {
		try {
			detail = await api.debt(id);
			nominalBayar = detail.remaining;
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal membuka rincian.', 'error');
		}
	}

	async function bayar() {
		if (!detail) return;
		memproses = true;
		try {
			const hasil = await api.createDebtPayment(detail.id, nominalBayar);
			detail = hasil;
			nominalBayar = hasil.remaining;
			toast(hasil.status === 'settled' ? `${hasil.person} sudah lunas. Meong!` : 'Pembayaran tercatat.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mencatat pembayaran.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function tambah() {
		memproses = true;
		try {
			await api.createDebt({
				person: formBaru.person,
				direction: arah,
				amount: formBaru.amount,
				note: formBaru.note,
				due_date: formBaru.due_date || null
			});
			tambahTerbuka = false;
			formBaru = { person: '', amount: 0, note: '', due_date: '' };
			toast(arah === 'hutang' ? 'Hutang baru dicatat.' : 'Piutang baru dicatat.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan catatan.', 'error');
		} finally {
			memproses = false;
		}
	}
</script>

<svelte:head>
	<title>Hutang piutang — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Hutang piutang</h1>
		<button type="button" class="btn btn-primary btn-sm min-h-11" onclick={() => (tambahTerbuka = true)}>
			Catat baru
		</button>
	</header>

	<div class="grid grid-cols-2 gap-2" role="tablist" aria-label="Pilih jenis catatan">
		<button
			type="button"
			role="tab"
			aria-selected={arah === 'hutang'}
			class="btn min-h-11 {arah === 'hutang' ? 'btn-primary' : 'btn-outline'}"
			onclick={() => (arah = 'hutang')}
		>
			Hutangku {formatRupiah(total.hutang)}
		</button>
		<button
			type="button"
			role="tab"
			aria-selected={arah === 'piutang'}
			class="btn min-h-11 {arah === 'piutang' ? 'btn-primary' : 'btn-outline'}"
			onclick={() => (arah = 'piutang')}
		>
			Piutangku {formatRupiah(total.piutang)}
		</button>
	</div>

	<label class="flex min-h-11 cursor-pointer items-center gap-3 rounded-xl border border-base-300 px-3">
		<input type="checkbox" class="checkbox" bind:checked={tampilLunas} onchange={() => void muat()} />
		<span class="text-sm">Tampilkan yang sudah lunas</span>
	</label>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else if terlihat.length === 0}
		<StateBlock
			state="empty"
			title={arah === 'hutang' ? 'Tidak ada hutang berjalan' : 'Tidak ada piutang berjalan'}
			message={'Kalau ada yang pinjam uang, catat di sini biar tidak lupa. Contoh obrolan: "gua minjem 200rb ke Budi".'}
			actionLabel="Catat baru"
			onaction={() => (tambahTerbuka = true)}
		/>
	{:else}
		<ul class="space-y-2">
			{#each terlihat as item (item.id)}
				<li class="card bg-base-200">
					<button
						type="button"
						class="w-full rounded-box text-left transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
						onclick={() => void bukaDetail(item.id)}
					>
						<div class="card-body gap-2 p-4">
							<div class="flex items-baseline justify-between gap-3">
								<span class="font-medium">{item.person}</span>
								<span class="font-display text-base font-semibold tabular-nums">
									{formatRupiah(item.remaining)}
								</span>
							</div>

							{#if item.note}
								<p class="text-xs opacity-70">{item.note}</p>
							{/if}

							<Progress
								value={item.amount_total > 0 ? (item.paid * 100) / item.amount_total : 0}
								status={item.status === 'settled' ? 'ok' : 'none'}
								label={`Pembayaran ${item.person}`}
							/>

							<p class="text-xs {item.overdue ? 'font-semibold text-error-ink' : 'opacity-70'}">
								{#if item.status === 'settled'}
									Lunas
								{:else if item.due_date}
									{item.overdue ? 'Telat sejak' : 'Jatuh tempo'} {labelTanggalPendek(item.due_date)}
								{:else}
									Tanpa jatuh tempo
								{/if}
								· Dibayar {formatRupiah(item.paid)} dari {formatRupiah(item.amount_total)}
							</p>
						</div>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Sheet
	open={detail !== null}
	title={detail ? `${detail.direction === 'hutang' ? 'Hutang ke' : 'Piutang dari'} ${detail.person}` : 'Rincian'}
	description={detail && detail.status === 'settled' ? 'Catatan ini sudah lunas.' : 'Catat pembayaran atau cicilannya di sini.'}
	onclose={() => (detail = null)}
>
	{#if detail}
		<div class="space-y-4">
			<dl class="grid grid-cols-2 gap-3 text-sm">
				<div>
					<dt class="opacity-70">Total</dt>
					<dd class="font-display text-lg font-semibold tabular-nums">{formatRupiah(detail.amount_total)}</dd>
				</div>
				<div class="text-right">
					<dt class="opacity-70">Sisa</dt>
					<dd class="font-display text-lg font-semibold tabular-nums">{formatRupiah(detail.remaining)}</dd>
				</div>
			</dl>

			{#if detail.note}
				<p class="rounded-xl bg-base-300/50 px-3 py-2 text-sm">{detail.note}</p>
			{/if}

			{#if detail.due_date}
				<p class="text-sm opacity-80">Jatuh tempo {labelTanggal(detail.due_date)}</p>
			{/if}

			{#if detail.payments && detail.payments.length > 0}
				<div>
					<h3 class="mb-2 text-sm font-semibold">Riwayat pembayaran</h3>
					<ul class="space-y-1 text-sm">
						{#each detail.payments as bayarItem (bayarItem.id)}
							<li class="flex items-baseline justify-between gap-3 rounded-xl bg-base-300/40 px-3 py-2">
								<span>{labelTanggal(bayarItem.paid_at)}</span>
								<span class="tabular-nums">{formatRupiah(bayarItem.amount)}</span>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			{#if detail.status !== 'settled'}
				<form
					class="space-y-3"
					onsubmit={(event) => {
						event.preventDefault();
						void bayar();
					}}
				>
					<div>
						<label class="label pb-1 text-sm font-semibold" for="nominal-bayar-hutang">Nominal pembayaran</label>
						<MoneyInput id="nominal-bayar-hutang" value={nominalBayar} onchange={(value) => (nominalBayar = value)} />
					</div>
					<div class="flex justify-end gap-2">
						<button type="button" class="btn btn-ghost min-h-11" onclick={() => (nominalBayar = detail?.remaining ?? 0)}>
							Isi sisa penuh
						</button>
						<button type="submit" class="btn btn-primary min-h-11" disabled={memproses || nominalBayar <= 0}>
							{memproses ? 'Menyimpan...' : 'Catat pembayaran'}
						</button>
					</div>
				</form>
			{/if}
		</div>
	{/if}
</Sheet>

<Sheet
	open={tambahTerbuka}
	title={arah === 'hutang' ? 'Catat hutang' : 'Catat piutang'}
	description={arah === 'hutang' ? 'Kamu yang berhutang ke orang lain.' : 'Orang lain yang berhutang ke kamu.'}
	onclose={() => (tambahTerbuka = false)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void tambah();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nama-orang">Nama orang</label>
			<input id="nama-orang" class="input input-bordered min-h-11 w-full" bind:value={formBaru.person} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nominal-hutang">Nominal</label>
			<MoneyInput id="nominal-hutang" value={formBaru.amount} onchange={(value) => (formBaru.amount = value)} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="catatan-hutang">Catatan</label>
			<input id="catatan-hutang" class="input input-bordered min-h-11 w-full" bind:value={formBaru.note} />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="jatuh-tempo">Jatuh tempo (opsional)</label>
			<input id="jatuh-tempo" type="date" class="input input-bordered min-h-11 w-full" bind:value={formBaru.due_date} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (tambahTerbuka = false)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={memproses || formBaru.amount <= 0 || !formBaru.person.trim()}>
				{memproses ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
