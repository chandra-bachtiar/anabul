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
	import { getarRingan } from '#lib/haptic';
	import { session } from '#lib/session.svelte';
	import {
		bulanIni,
		formatRupiah,
		geserBulan,
		labelBulan,
		labelHari,
		labelTanggalPendek,
		statusBudgetLabel
	} from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Dashboard, Need } from '#lib/types';

	let bulan = $state(bulanIni());
	let data = $state<Dashboard | null>(null);
	let memuat = $state(true);
	let pesanError = $state('');

	let perluDibayar = $state<Need | null>(null);
	let nominalBayar = $state(0);
	let perayaan = $state(0);

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			data = await api.dashboard(bulan);
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat data bulan ini.';
		} finally {
			memuat = false;
		}
	}

	$effect(() => {
		bulan;
		void muat();
	});

	onMount(() => onChatChange(() => void muat()));

	function gantiBulan(delta: number) {
		bulan = geserBulan(bulan, delta);
	}

	function bukaBayar(need: Need) {
		perluDibayar = need;
		nominalBayar = need.amount;
	}

	async function bayar() {
		if (!perluDibayar) return;
		const need = perluDibayar;
		perluDibayar = null;
		try {
			await api.payNeed(need.id, nominalBayar);
			perayaan += 1;
			getarRingan();
			toast(`${need.name} ditandai lunas dan masuk pengeluaran.`);
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menandai lunas.', 'error');
		}
	}

	const bulanBerjalan = $derived(bulan === bulanIni());
</script>

<svelte:head>
	<title>ANABUL — Asisten Ngatur Anggaran Bulanan</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-3">
		<div class="flex items-center gap-3">
			<Mascot mood={data?.mood ?? 'senang'} size={56} />
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
			onclick={() => gantiBulan(-1)}
			aria-label="Lihat bulan sebelumnya"
		>
			<svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" aria-hidden="true">
				<path d="M14.5 6.5 9 12l5.5 5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</button>

		<div class="flex items-center gap-2">
			<span class="font-display text-base font-semibold">{labelBulan(bulan)}</span>
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
			{#if !bulanBerjalan}
				<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => (bulan = bulanIni())}>
					Bulan ini
				</button>
			{/if}
			<button
				type="button"
				class="btn btn-ghost btn-sm min-h-11 px-2"
				onclick={() => gantiBulan(1)}
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
					<Mascot mood="khawatir" size={44} alt="" />
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

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else if data}
		{#if data.locked}
			<div class="card border border-warning/50 bg-warning/10">
				<div class="card-body flex-row items-center gap-3 p-4">
					<Mascot mood="tidur" size={44} alt="" />
					<p class="text-sm">
						Bulan ini sudah ditutup, jadi catatan tidak bisa diubah. Buka kuncinya dari halaman
						<a class="link font-semibold" href="/tutup-bulan">Tutup bulan</a> kalau perlu koreksi.
					</p>
				</div>
			</div>
		{/if}

		<!-- Fokus utama halaman: sisa budget bulan ini. -->
		<section class="card bg-base-200" aria-labelledby="judul-budget">
			<div class="card-body gap-3">
				<div class="flex items-start justify-between gap-3">
					<div>
						<h2 id="judul-budget" class="text-sm font-semibold opacity-70">Sisa budget bulan ini</h2>
						{#key data.budget_remaining}
							<p class="anim-pop font-display text-4xl leading-tight font-semibold tabular-nums">
								{formatRupiah(data.budget_remaining)}
							</p>
						{/key}
					</div>
					{#if data.budget_status === 'warn' || data.budget_status === 'over'}
						<Mascot mood="khawatir" size={64} />
					{/if}
				</div>

				{#if data.budget_total === 0}
					<div class="space-y-2">
						<p class="text-sm opacity-80">Belum ada budget bulan ini, jadi angka di atas masih kosong.</p>
						<a class="btn btn-outline btn-sm min-h-11" href="/budget">Atur budget {labelBulan(bulan)}</a>
					</div>
				{:else}
					<Progress
						value={data.budget_total > 0 ? (data.budget_spent * 100) / data.budget_total : 0}
						status={data.budget_status}
						label={`Pemakaian budget ${labelBulan(bulan)}`}
						tebal
					/>
					<p class="text-sm">
						{statusBudgetLabel(data.budget_status)}. Terpakai {formatRupiah(data.budget_spent)} dari
						{formatRupiah(data.budget_total)}.
					</p>
				{/if}
			</div>
		</section>

		<!-- Uang masuk, uang keluar, selisih: satu kartu karena dibaca berurutan. -->
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
				<ul class="anim-daftar space-y-2">
					{#each data.needs as need (need.id)}
						<li>
							<button
								type="button"
								class="card w-full bg-base-200 text-left transition-colors hover:bg-base-300/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
								onclick={() => bukaBayar(need)}
							>
								<div class="card-body flex-row items-center justify-between gap-3 p-4">
									<div>
										<p class="font-medium">{need.name}</p>
										<p class="text-xs {need.overdue ? 'font-semibold text-error-ink' : 'opacity-70'}">
											{need.overdue ? 'Telat sejak' : 'Jatuh tempo'} {labelTanggalPendek(need.due_date)}
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
			<section class="card bg-base-200" aria-labelledby="judul-kategori">
				<div class="card-body gap-3 p-4">
					<div class="flex items-baseline justify-between gap-2">
						<h2 id="judul-kategori" class="font-display text-base font-semibold">Paling banyak dipakai</h2>
						<a class="link inline-block py-1.5 text-xs font-semibold" href="/budget">Atur budget</a>
					</div>

					<ul class="space-y-3">
						{#each data.top_categories as kategori (kategori.category_id ?? kategori.category_name)}
							<li class="space-y-1">
								<div class="flex items-baseline justify-between gap-3 text-sm">
									<span class="font-medium">{kategori.category_name}</span>
									<span class="tabular-nums opacity-70">
										{formatRupiah(kategori.spent)}
										{#if kategori.budget > 0}
											<span>/ {formatRupiah(kategori.budget)}</span>
										{/if}
									</span>
								</div>
								<Progress
									value={kategori.progress}
									status={kategori.status}
									label={`Pemakaian ${kategori.category_name}`}
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
				<ul class="anim-daftar space-y-2">
					{#each data.recent as transaksi (transaksi.id)}
						<li class="card bg-base-200">
							<div class="card-body flex-row items-center justify-between gap-3 p-4">
								<div class="min-w-0">
									<p class="truncate font-medium">{transaksi.note || 'Tanpa catatan'}</p>
									<p class="text-xs opacity-70">
										{labelHari(transaksi.occurred_at)}
										{#if transaksi.category_name}
											· {transaksi.category_name}
										{/if}
									</p>
								</div>
								<span
									class="font-display text-base font-semibold tabular-nums {transaksi.type === 'income'
										? 'text-success-ink'
										: 'text-error-ink'}"
								>
									{transaksi.type === 'income' ? '+' : '-'}{formatRupiah(transaksi.amount)}
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

<Confetti trigger={perayaan} />

<Sheet
	open={perluDibayar !== null}
	title={perluDibayar ? `Bayar ${perluDibayar.name}` : 'Bayar kebutuhan'}
	description="Nominal bisa diubah kalau tagihannya beda dari perkiraan."
	onclose={() => (perluDibayar = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void bayar();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nominal-bayar">Nominal</label>
			<MoneyInput id="nominal-bayar" value={nominalBayar} onchange={(value) => (nominalBayar = value)} />
		</div>
		<div class="flex flex-wrap justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (perluDibayar = null)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={nominalBayar <= 0}>
				Tandai lunas
			</button>
		</div>
	</form>
</Sheet>
