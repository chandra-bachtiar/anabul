<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import ChatPanel from '#lib/components/ChatPanel.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Progress from '#lib/components/Progress.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { onChatChange } from '#lib/chat.svelte';
	import Confetti from '#lib/components/Confetti.svelte';
	import { hapticLight } from '#lib/haptic';
	import { session } from '#lib/session.svelte';
	import {
		currentMonth,
		formatRupiah,
		shiftMonth,
		formatMonth,
		formatDay,
		formatDateShort,
		budgetStatusLabel
	} from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Dashboard, Need } from '#lib/types';

	let month = $state(currentMonth());
	let data = $state<Dashboard | null>(null);
	let loading = $state(true);
	let errorMessage = $state('');

	let needToPay = $state<Need | null>(null);
	let paymentAmount = $state(0);
	let celebration = $state(0);

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			data = await api.dashboard(month);
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat data bulan ini.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		month;
		void load();
	});

	onMount(() => onChatChange(() => void load()));

	function changeMonth(delta: number) {
		month = shiftMonth(month, delta);
	}

	function openPayment(need: Need) {
		needToPay = need;
		paymentAmount = need.amount;
	}

	async function pay() {
		if (!needToPay) return;
		const need = needToPay;
		needToPay = null;
		try {
			await api.payNeed(need.id, paymentAmount);
			celebration += 1;
			hapticLight();
			toast(`${need.name} ditandai lunas dan masuk pengeluaran.`);
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menandai lunas.', 'error');
		}
	}

	const isCurrentMonth = $derived(month === currentMonth());
</script>

<svelte:head>
	<title>ANABUL — Asisten Ngatur Anggaran Bulanan</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-3">
		<div class="flex items-center gap-3">
			<Mascot mood={data?.mood ?? 'happy'} size={56} />
			<div>
				<h1 class="font-display text-2xl font-semibold tracking-wide text-primary-ink">ANABUL</h1>
				<p class="text-xs opacity-70">Asisten Ngatur Anggaran Bulanan</p>
			</div>
		</div>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button
			type="button"
			class="btn btn-ghost btn-sm min-h-11 px-2"
			onclick={() => changeMonth(-1)}
			aria-label="Lihat bulan sebelumnya"
		>
			<svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" aria-hidden="true">
				<path d="M14.5 6.5 9 12l5.5 5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</button>

		<div class="flex items-center gap-2">
			<span class="font-display text-base font-semibold">{formatMonth(month)}</span>
			{#if data?.locked}
				<span class="badge badge-warning badge-sm gap-1 font-semibold">
					<svg viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" aria-hidden="true">
						<path d="M7.5 10.5V8.2a4.5 4.5 0 0 1 9 0v2.3" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
						<rect x="5.6" y="10.4" width="12.8" height="9" rx="2.4" stroke="currentColor" stroke-width="2" />
					</svg>
					Terkunci
				</span>
			{/if}
		</div>

		<div class="flex items-center gap-1">
			{#if !isCurrentMonth}
				<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => (month = currentMonth())}>
					Bulan ini
				</button>
			{/if}
			<button
				type="button"
				class="btn btn-ghost btn-sm min-h-11 px-2"
				onclick={() => changeMonth(1)}
				aria-label="Lihat bulan berikutnya"
			>
				<svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" aria-hidden="true">
					<path d="M9.5 6.5 15 12l-5.5 5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
				</svg>
			</button>
		</div>
	</div>

	{#if session.usingDefaultPassword}
		<div class="card border border-warning/60 bg-warning/10">
			<div class="card-body gap-3 p-4">
				<div class="flex items-start gap-3">
					<Mascot mood="worried" size={44} alt="" />
					<div>
						<h2 class="font-display text-base font-semibold">Passwordnya masih default</h2>
						<p class="mt-1 text-sm">
							Password kamu sekarang <strong>anabul123</strong>, sama dengan yang tertulis di README.
							Ganti dulu supaya catatan keuanganmu tidak bisa dibuka orang lain.
						</p>
					</div>
				</div>
				<a class="btn btn-primary btn-sm min-h-11 self-start" href="/lainnya/pengaturan#ganti-password">
					Ganti password sekarang
				</a>
			</div>
		</div>
	{/if}

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else if data}
		{#if data.locked}
			<div class="card border border-warning/50 bg-warning/10">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="asleep" size={44} alt="" />
					<p class="text-sm">
						Bulan ini sudah ditutup, jadi catatan tidak bisa diubah. Buka kuncinya dari halaman
						<a class="link font-semibold" href="/tutup-bulan">Tutup bulan</a> kalau perlu koreksi.
					</p>
				</div>
			</div>
		{/if}

		<section class="card bg-base-200" aria-labelledby="budget-title">
			<div class="card-body gap-3">
				<div class="flex items-start justify-between gap-3">
					<div>
						<h2 id="budget-title" class="text-sm font-semibold opacity-70">Sisa budget bulan ini</h2>
						{#key data.budget_remaining}
							<p class="anim-pop font-display text-4xl leading-tight font-semibold tabular-nums">
								{formatRupiah(data.budget_remaining)}
							</p>
						{/key}
					</div>
					{#if data.budget_status === 'warn' || data.budget_status === 'over'}
						<Mascot mood="worried" size={64} />
					{/if}
				</div>

				{#if data.budget_total === 0}
					<div class="space-y-2">
						<p class="text-sm opacity-80">Belum ada budget bulan ini, jadi angka di atas masih kosong.</p>
						<a class="btn btn-outline btn-sm min-h-11" href="/budget">Atur budget {formatMonth(month)}</a>
					</div>
				{:else}
					<Progress
						value={data.budget_total > 0 ? (data.budget_spent * 100) / data.budget_total : 0}
						status={data.budget_status}
						label={`Pemakaian budget ${formatMonth(month)}`}
						thick
					/>
					<p class="text-sm">
						{budgetStatusLabel(data.budget_status)}. Terpakai {formatRupiah(data.budget_spent)} dari
						{formatRupiah(data.budget_total)}.
					</p>
				{/if}
			</div>
		</section>

		<!-- One card: income, expense and net are read in sequence. -->
		<section class="card bg-base-200" aria-label="Ringkasan uang bulan ini">
			<div class="card-body grid grid-cols-3 gap-2 p-4">
				<div>
					<p class="text-xs font-semibold opacity-70">Masuk</p>
					<p class="font-display text-lg font-semibold tabular-nums text-success-ink">
						{formatRupiah(data.income)}
					</p>
				</div>
				<div class="border-x border-base-300 px-2 text-center">
					<p class="text-xs font-semibold opacity-70">Keluar</p>
					<p class="font-display text-lg font-semibold tabular-nums text-error-ink">
						{formatRupiah(data.expense)}
					</p>
				</div>
				<div class="text-right">
					<p class="text-xs font-semibold opacity-70">Selisih</p>
					<p class="font-display text-lg font-semibold tabular-nums {data.net < 0 ? 'text-error-ink' : ''}">
						{formatRupiah(data.net)}
					</p>
				</div>
			</div>
		</section>

		<section aria-labelledby="judul-kebutuhan">
			<div class="mb-2 flex items-baseline justify-between gap-2">
				<h2 id="judul-kebutuhan" class="font-display text-base font-semibold">Kebutuhan bulan ini</h2>
				<a class="link inline-block py-1.5 text-xs font-semibold" href="/lainnya/kebutuhan">Kelola kebutuhan</a>
			</div>

			{#if data.needs.length === 0}
				<div class="card bg-base-200">
					<div class="card-body p-4">
						<p class="text-sm">
							{#if data.needs_unpaid === 0}
								Semua kebutuhan bulan ini sudah lunas. Meong!
							{:else}
								Belum ada kebutuhan yang jatuh tempo bulan ini.
							{/if}
						</p>
					</div>
				</div>
			{:else}
				<ul class="anim-list space-y-2">
					{#each data.needs as need (need.id)}
						<li>
							<button
								type="button"
								class="card w-full bg-base-200 text-left transition-colors hover:bg-base-300/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
								onclick={() => openPayment(need)}
							>
								<div class="card-body flex-row items-center justify-between gap-3 p-4">
									<div>
										<p class="font-medium">{need.name}</p>
										<p class="text-xs {need.overdue ? 'font-semibold text-error-ink' : 'opacity-70'}">
											{need.overdue ? 'Telat sejak' : 'Jatuh tempo'} {formatDateShort(need.due_date)}
										</p>
									</div>
									<span class="font-display text-base font-semibold tabular-nums">
										{formatRupiah(need.amount)}
									</span>
								</div>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		{#if data.debts.hutang > 0 || data.debts.piutang > 0}
			<section class="card bg-base-200" aria-label="Hutang piutang">
				<div class="card-body grid grid-cols-2 gap-3 p-4">
					<div>
						<p class="text-xs font-semibold opacity-70">Hutangku</p>
						<p class="font-display text-lg font-semibold tabular-nums">{formatRupiah(data.debts.hutang)}</p>
					</div>
					<div class="text-right">
						<p class="text-xs font-semibold opacity-70">Piutangku</p>
						<p class="font-display text-lg font-semibold tabular-nums">{formatRupiah(data.debts.piutang)}</p>
					</div>
					<a class="link col-span-2 inline-block py-1.5 text-xs font-semibold" href="/hutang">Lihat rinciannya</a>
				</div>
			</section>
		{/if}

		{#if data.top_categories.length > 0}
			<section class="card bg-base-200" aria-labelledby="category-title">
				<div class="card-body gap-3 p-4">
					<div class="flex items-baseline justify-between gap-2">
						<h2 id="category-title" class="font-display text-base font-semibold">Paling banyak dipakai</h2>
						<a class="link inline-block py-1.5 text-xs font-semibold" href="/budget">Atur budget</a>
					</div>

					<ul class="space-y-3">
						{#each data.top_categories as category (category.category_id ?? category.category_name)}
							<li class="space-y-1">
								<div class="flex items-baseline justify-between gap-3 text-sm">
									<span class="font-medium">{category.category_name}</span>
									<span class="tabular-nums opacity-70">
										{formatRupiah(category.spent)}
										{#if category.budget > 0}
											<span>/ {formatRupiah(category.budget)}</span>
										{/if}
									</span>
								</div>
								<Progress
									value={category.progress}
									status={category.status}
									label={`Pemakaian ${category.category_name}`}
								/>
							</li>
						{/each}
					</ul>
				</div>
			</section>
		{/if}

		<section aria-labelledby="judul-terakhir">
			<div class="mb-2 flex items-baseline justify-between gap-2">
				<h2 id="judul-terakhir" class="font-display text-base font-semibold">Catatan terakhir</h2>
				<a class="link inline-block py-1.5 text-xs font-semibold" href="/transaksi">Semua transaksi</a>
			</div>

			{#if data.recent.length === 0}
				<StateBlock
					state="empty"
					title="Belum ada catatan bulan ini"
					message={'Ketik "kopi 25rb" di kolom obrolan bawah, nanti ANABUL catat sendiri.'}
				/>
			{:else}
				<ul class="anim-list space-y-2">
					{#each data.recent as transaction (transaction.id)}
						<li class="card bg-base-200">
							<div class="card-body flex-row items-center justify-between gap-3 p-4">
								<div class="min-w-0">
									<p class="truncate font-medium">{transaction.note || 'Tanpa catatan'}</p>
									<p class="text-xs opacity-70">
										{formatDay(transaction.occurred_at)}
										{#if transaction.category_name}
											· {transaction.category_name}
										{/if}
									</p>
								</div>
								<span
									class="font-display text-base font-semibold tabular-nums {transaction.type === 'income'
										? 'text-success-ink'
										: 'text-error-ink'}"
								>
									{transaction.type === 'income' ? '+' : '-'}{formatRupiah(transaction.amount)}
								</span>
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

	<ChatPanel />
</div>

<Confetti trigger={celebration} />

<Sheet
	open={needToPay !== null}
	title={needToPay ? `Bayar ${needToPay.name}` : 'Bayar kebutuhan'}
	description="Nominal bisa diubah kalau tagihannya beda dari perkiraan."
	onclose={() => (needToPay = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void pay();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="payment-amount">Nominal</label>
			<MoneyInput id="payment-amount" value={paymentAmount} onchange={(value) => (paymentAmount = value)} />
		</div>
		<div class="flex flex-wrap justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (needToPay = null)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={paymentAmount <= 0}>
				Tandai lunas
			</button>
		</div>
	</form>
</Sheet>
