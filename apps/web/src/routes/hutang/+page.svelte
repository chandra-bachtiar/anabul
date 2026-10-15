<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Confetti from '#lib/components/Confetti.svelte';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { formatRupiah, formatDate, formatDateShort } from '#lib/format';
	import { hapticLight, hapticCelebrate } from '#lib/haptic';
	import { toast } from '#lib/toast.svelte';
	import type { Debt, DebtTotals } from '#lib/types';

	let direction = $state<'hutang' | 'piutang'>('hutang');
	let debts = $state<Debt[]>([]);
	let total = $state<DebtTotals>({ hutang: 0, piutang: 0 });
	let loading = $state(true);
	let errorMessage = $state('');
	let showSettled = $state(false);

	let detail = $state<Debt | null>(null);
	let paymentAmount = $state(0);
	let recordTransaction = $state(true);
	let processing = $state(false);
	let celebration = $state(0);

	let addSheetOpen = $state(false);
	let newForm = $state({ person: '', amount: 0, note: '', due_date: '' });

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			const result = await api.debts(showSettled ? undefined : 'open');
			debts = result.debts;
			total = result.totals;
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat hutang piutang.';
		} finally {
			loading = false;
		}
	}

	onMount(() => void load());

	const visibleItems = $derived(debts.filter((item) => item.direction === direction));

	async function openDetail(id: number) {
		try {
			detail = await api.debt(id);
			paymentAmount = detail.remaining;
			recordTransaction = true;
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal membuka rincian.', 'error');
		}
	}

	async function pay() {
		if (!detail) return;
		const person = detail.person;
		const includeInJournal = recordTransaction;
		const isDebt = detail.direction === 'hutang';
		processing = true;
		try {
			const result = await api.createDebtPayment(detail.id, paymentAmount, includeInJournal);

			detail = null;
			paymentAmount = 0;

			const journalNote = includeInJournal
				? isDebt
					? ' Masuk pengeluaran bulan ini.'
					: ' Masuk pemasukan bulan ini.'
				: ' Tidak masuk catatan harian.';

			if (result.status === 'settled') {
				celebration += 1;
				hapticCelebrate();
				toast(`${person} sudah lunas. Meong!${journalNote}`, 'ok', 6000);
			} else {
				hapticLight();
				toast(`Pembayaran ${person} tercatat. Sisa ${formatRupiah(result.remaining)}.${journalNote}`, 'ok', 5000);
			}
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mencatat pembayaran.', 'error');
		} finally {
			processing = false;
		}
	}

	async function createDebt() {
		processing = true;
		try {
			await api.createDebt({
				person: newForm.person,
				direction,
				amount: newForm.amount,
				note: newForm.note,
				due_date: newForm.due_date || null
			});
			addSheetOpen = false;
			newForm = { person: '', amount: 0, note: '', due_date: '' };
			hapticLight();
			toast(direction === 'hutang' ? 'Hutang baru dicatat.' : 'Piutang baru dicatat.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan catatan.', 'error');
		} finally {
			processing = false;
		}
	}

	function dueStatus(item: Debt): string {
		if (item.status === 'settled') return 'Lunas';
		if (!item.due_date) return 'Tanpa jatuh tempo';
		return `${item.overdue ? 'Telat sejak' : 'Jatuh tempo'} ${formatDateShort(item.due_date)}`;
	}
</script>

<svelte:head>
	<title>Hutang piutang — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Hutang piutang</h1>
		<button type="button" class="btn btn-primary btn-sm min-h-11" onclick={() => (addSheetOpen = true)}>
			Catat baru
		</button>
	</header>

	<div class="grid grid-cols-2 gap-2" role="tablist" aria-label="Pilih jenis catatan">
		<button
			type="button"
			role="tab"
			aria-selected={direction === 'hutang'}
			class="btn min-h-11 {direction === 'hutang' ? 'btn-primary' : 'btn-outline'}"
			onclick={() => (direction = 'hutang')}
		>
			Hutangku {formatRupiah(total.hutang)}
		</button>
		<button
			type="button"
			role="tab"
			aria-selected={direction === 'piutang'}
			class="btn min-h-11 {direction === 'piutang' ? 'btn-primary' : 'btn-outline'}"
			onclick={() => (direction = 'piutang')}
		>
			Piutangku {formatRupiah(total.piutang)}
		</button>
	</div>

	<label class="flex min-h-11 cursor-pointer items-center gap-3 rounded-xl border border-base-300 px-3">
		<input type="checkbox" class="checkbox" bind:checked={showSettled} onchange={() => void load()} />
		<span class="text-sm">Tampilkan yang sudah lunas</span>
	</label>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else if visibleItems.length === 0}
		<StateBlock
			state="empty"
			title={direction === 'hutang' ? 'Tidak ada hutang berjalan' : 'Tidak ada piutang berjalan'}
			message={'Kalau ada yang pinjam uang, catat di sini biar tidak lupa. Contoh obrolan: "gua minjem 200rb ke Budi".'}
			actionLabel="Catat baru"
			onaction={() => (addSheetOpen = true)}
		/>
	{:else}
		<ul class="anim-list space-y-2">
			{#each visibleItems as item (item.id)}
				<li class="card card-hover-lift bg-base-200">
					<button
						type="button"
						class="w-full rounded-box text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
						onclick={() => void openDetail(item.id)}
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
								{dueStatus(item)}
							</p>

							<p class="text-xs opacity-70">
								{#if item.payment_count > 0}
									{item.payment_count}x bayar
									{#if item.last_payment_at}· terakhir {formatDateShort(item.last_payment_at)}{/if}
									· total {formatRupiah(item.paid)} dari {formatRupiah(item.amount_total)}
								{:else}
									Belum ada pembayaran · ketuk untuk lihat riwayat dan catat bayar
								{/if}
							</p>
						</div>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Confetti trigger={celebration} />

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
				<p class="text-sm opacity-80">Jatuh tempo {formatDate(detail.due_date)}</p>
			{/if}

			<div>
				<h3 class="mb-2 flex items-baseline justify-between gap-2 text-sm font-semibold">
					<span>Riwayat pembayaran</span>
					<span class="tabular-nums opacity-70">{detail.payment_count}x · {formatRupiah(detail.paid)}</span>
				</h3>
				{#if detail.payments && detail.payments.length > 0}
					<ul class="anim-list space-y-1 text-sm">
						{#each detail.payments as payment, index (payment.id)}
							<li class="flex items-baseline justify-between gap-3 rounded-xl bg-base-300/40 px-3 py-2">
								<span class="opacity-70">
									{#if index === 0}Terakhir · {/if}{formatDate(payment.paid_at)}
								</span>
								<span class="tabular-nums font-medium">{formatRupiah(payment.amount)}</span>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="rounded-xl bg-base-300/40 px-3 py-2 text-sm opacity-70">
						Belum ada pembayaran tercatat. Setiap cicilan yang kamu simpan akan muncul di sini.
					</p>
				{/if}
			</div>

			{#if detail.status !== 'settled'}
				<form
					class="space-y-3"
					onsubmit={(event) => {
						event.preventDefault();
						void pay();
					}}
				>
					<div>
						<label class="label pb-1 text-sm font-semibold" for="debt-payment-amount">Nominal pembayaran</label>
						<MoneyInput id="debt-payment-amount" value={paymentAmount} onchange={(value) => (paymentAmount = value)} />
					</div>

					<label class="flex min-h-11 cursor-pointer items-start gap-3 rounded-xl border border-base-300 px-3 py-2">
						<input type="checkbox" class="checkbox mt-1" bind:checked={recordTransaction} />
						<span class="text-sm">
							Catat juga di buku harian
							<span class="block text-xs opacity-70">
								{detail.direction === 'hutang'
									? 'Uang yang keluar dihitung sebagai pengeluaran bulan ini.'
									: 'Uang yang diterima dihitung sebagai pemasukan bulan ini.'}
							</span>
						</span>
					</label>

					<div class="flex justify-end gap-2">
						<button type="button" class="btn btn-ghost min-h-11" onclick={() => (paymentAmount = detail?.remaining ?? 0)}>
							Isi sisa penuh
						</button>
						<button type="submit" class="btn btn-primary min-h-11" disabled={processing || paymentAmount <= 0}>
							{processing ? 'Menyimpan...' : 'Catat pembayaran'}
						</button>
					</div>
				</form>
			{/if}
		</div>
	{/if}
</Sheet>

<Sheet
	open={addSheetOpen}
	title={direction === 'hutang' ? 'Catat hutang' : 'Catat piutang'}
	description={direction === 'hutang' ? 'Kamu yang berhutang ke orang lain.' : 'Orang lain yang berhutang ke kamu.'}
	onclose={() => (addSheetOpen = false)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void createDebt();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nama-orang">Nama orang</label>
			<input id="nama-orang" class="input input-bordered min-h-11 w-full" bind:value={newForm.person} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nominal-hutang">Nominal</label>
			<MoneyInput id="nominal-hutang" value={newForm.amount} onchange={(value) => (newForm.amount = value)} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="catatan-hutang">Catatan</label>
			<input id="catatan-hutang" class="input input-bordered min-h-11 w-full" bind:value={newForm.note} />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="jatuh-tempo">Jatuh tempo (opsional)</label>
			<input id="jatuh-tempo" type="date" class="input input-bordered min-h-11 w-full" bind:value={newForm.due_date} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (addSheetOpen = false)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={processing || newForm.amount <= 0 || !newForm.person.trim()}>
				{processing ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
