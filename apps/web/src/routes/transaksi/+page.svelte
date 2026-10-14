<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { onChatChange } from '#lib/chat.svelte';
	import { getarRingan } from '#lib/haptic';
	import {
		bulanDariTanggal,
		bulanIni,
		formatRupiah,
		geserBulan,
		isoDariTanggal,
		labelBulan,
		labelHari,
		labelJam,
		tanggalInput,
		tanggalKemarin
	} from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Account, Category, Transaction } from '#lib/types';

	let bulan = $state(bulanIni());
	let daftar = $state<Transaction[]>([]);
	let kategori = $state<Category[]>([]);
	let akun = $state<Account[]>([]);
	let kursor = $state<number | null>(null);
	let memuat = $state(true);
	let memuatLagi = $state(false);
	let pesanError = $state('');

	let pencarian = $state('');
	let filterKategori = $state<number | 0>(0);

	let editor = $state<Transaction | null>(null);
	let form = $state({
		type: 'expense',
		amount: 0,
		note: '',
		category_id: 0,
		account_id: 0,
		tanggal: tanggalInput()
	});
	let simpanProses = $state(false);

	let tambahTerbuka = $state(false);
	let formBaru = $state({
		type: 'expense',
		amount: 0,
		note: '',
		category_id: 0,
		account_id: 0,
		tanggal: tanggalInput()
	});

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			const hasil = await api.transactions({
				month: bulan,
				q: pencarian.trim() || undefined,
				category_id: filterKategori || undefined,
				limit: 60
			});
			daftar = hasil.transactions;
			kursor = hasil.next_cursor;
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat transaksi.';
		} finally {
			memuat = false;
		}
	}

	async function muatLagi() {
		if (!kursor || memuatLagi) return;
		memuatLagi = true;
		try {
			const hasil = await api.transactions({
				month: bulan,
				q: pencarian.trim() || undefined,
				category_id: filterKategori || undefined,
				cursor: kursor,
				limit: 60
			});
			daftar = [...daftar, ...hasil.transactions];
			kursor = hasil.next_cursor;
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal memuat lanjutan.', 'error');
		} finally {
			memuatLagi = false;
		}
	}

	onMount(() => {
		void (async () => {
			try {
				const [kategoriHasil, akunHasil] = await Promise.all([api.categories(), api.accounts()]);
				kategori = kategoriHasil.categories;
				akun = akunHasil.accounts;
			} catch {
				toast('Gagal memuat daftar kategori dan akun.', 'error');
			}
		})();

		return onChatChange(() => void muat());
	});

	$effect(() => {
		bulan;
		filterKategori;
		void muat();
	});

	let timerCari: ReturnType<typeof setTimeout> | undefined;
	function cariBerubah(nilai: string) {
		pencarian = nilai;
		clearTimeout(timerCari);
		timerCari = setTimeout(() => void muat(), 350);
	}

	const kelompok = $derived.by(() => {
		const hasil: { judul: string; items: Transaction[] }[] = [];
		for (const item of daftar) {
			const judul = labelHari(item.occurred_at);
			const terakhir = hasil[hasil.length - 1];
			if (terakhir && terakhir.judul === judul) {
				terakhir.items.push(item);
			} else {
				hasil.push({ judul, items: [item] });
			}
		}
		return hasil;
	});

	function bukaEditor(transaksi: Transaction) {
		editor = transaksi;
		form = {
			type: transaksi.type,
			amount: transaksi.amount,
			note: transaksi.note,
			category_id: transaksi.category_id ?? 0,
			account_id: transaksi.account_id ?? 0,
			tanggal: tanggalInput(transaksi.occurred_at)
		};
	}

	// Catatan yang tanggalnya pindah bulan tidak akan terlihat di daftar bulan
	// yang sedang dibuka, jadi pemakai diberi tahu tujuannya ke mana.
	function beriTahuPindahBulan(tanggal: string) {
		const bulanTujuan = bulanDariTanggal(tanggal);
		if (bulanTujuan !== bulan) {
			toast(`Tersimpan di ${labelBulan(bulanTujuan)}.`);
			return true;
		}
		return false;
	}

	async function simpanEdit() {
		if (!editor) return;
		simpanProses = true;
		try {
			await api.updateTransaction(editor.id, {
				type: form.type,
				amount: form.amount,
				note: form.note,
				category_id: form.category_id || null,
				account_id: form.account_id || null,
				clear_category: form.category_id === 0,
				clear_account: form.account_id === 0,
				// Jam aslinya dipertahankan, yang berubah hanya tanggalnya.
				occurred_at: isoDariTanggal(form.tanggal, editor.occurred_at)
			});
			const pindahBulan = beriTahuPindahBulan(form.tanggal);
			editor = null;
			getarRingan();
			if (!pindahBulan) toast('Perubahan tersimpan.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan perubahan.', 'error');
		} finally {
			simpanProses = false;
		}
	}

	async function hapus(transaksi: Transaction) {
		try {
			await api.deleteTransaction(transaksi.id);
			editor = null;
			toast('Transaksi dihapus.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus transaksi.', 'error');
		}
	}

	// Saat sedang melihat bulan lain, tanggal awal diarahkan ke bulan itu supaya
	// catatan baru tidak diam-diam masuk ke bulan sekarang.
	function tanggalAwal() {
		return bulan === bulanIni() ? tanggalInput() : `${bulan}-01`;
	}

	function bukaTambah() {
		formBaru = {
			type: 'expense',
			amount: 0,
			note: '',
			category_id: 0,
			account_id: 0,
			tanggal: tanggalAwal()
		};
		tambahTerbuka = true;
	}

	async function tambah() {
		simpanProses = true;
		try {
			await api.createTransaction({
				type: formBaru.type,
				amount: formBaru.amount,
				note: formBaru.note,
				category_id: formBaru.category_id || null,
				account_id: formBaru.account_id || null,
				occurred_at: isoDariTanggal(formBaru.tanggal)
			});
			const pindahBulan = beriTahuPindahBulan(formBaru.tanggal);
			tambahTerbuka = false;
			formBaru = {
				type: 'expense',
				amount: 0,
				note: '',
				category_id: 0,
				account_id: 0,
				tanggal: tanggalAwal()
			};
			getarRingan();
			if (!pindahBulan) toast('Catatan baru tersimpan.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan catatan.', 'error');
		} finally {
			simpanProses = false;
		}
	}
</script>

<svelte:head>
	<title>Transaksi — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Transaksi</h1>
		<button type="button" class="btn btn-primary btn-sm min-h-11" onclick={bukaTambah}>
			Tambah manual
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, -1))} aria-label="Bulan sebelumnya">
			‹
		</button>
		<span class="font-display text-base font-semibold">{labelBulan(bulan)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, 1))} aria-label="Bulan berikutnya">
			›
		</button>
	</div>

	<div class="flex flex-col gap-2 sm:flex-row">
		<div class="flex-1">
			<label class="sr-only" for="cari-transaksi">Cari catatan</label>
			<input
				id="cari-transaksi"
				type="search"
				class="input input-bordered min-h-11 w-full"
				placeholder="Cari catatan, mis. kopi"
				value={pencarian}
				oninput={(event) => cariBerubah(event.currentTarget.value)}
			/>
		</div>
		<div>
			<label class="sr-only" for="filter-kategori">Filter kategori</label>
			<select id="filter-kategori" class="select select-bordered min-h-11 w-full sm:w-44" bind:value={filterKategori}>
				<option value={0}>Semua kategori</option>
				{#each kategori as item (item.id)}
					<option value={item.id}>{item.name}</option>
				{/each}
			</select>
		</div>
	</div>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else if daftar.length === 0}
		<StateBlock
			state="empty"
			title="Belum ada transaksi di bulan ini"
			message="Coba ganti bulan, ubah filter, atau catat lewat obrolan di Home."
			actionLabel="Tambah manual"
			onaction={bukaTambah}
		/>
	{:else}
		<div class="space-y-4">
			{#each kelompok as grup (grup.judul)}
				<section aria-label={grup.judul}>
					<h2 class="mb-2 text-xs font-semibold tracking-wide uppercase opacity-70">{grup.judul}</h2>
					<ul class="anim-daftar space-y-2">
						{#each grup.items as transaksi (transaksi.id)}
							<li class="card bg-base-200">
								<button
									type="button"
									class="w-full rounded-box text-left transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
									onclick={() => bukaEditor(transaksi)}
								>
									<div class="card-body flex-row items-center justify-between gap-3 p-4">
										<div class="min-w-0">
											<p class="truncate font-medium">{transaksi.note || 'Tanpa catatan'}</p>
											<p class="text-xs opacity-70">
												{labelJam(transaksi.occurred_at)}
												{#if transaksi.category_name}· {transaksi.category_name}{/if}
												{#if transaksi.account_name}· {transaksi.account_name}{/if}
												{#if transaksi.source === 'telegram'}· dari Telegram{/if}
												{#if transaksi.source === 'need'}· dari kebutuhan{/if}
											</p>
										</div>
										<span class="font-display text-base font-semibold tabular-nums {transaksi.type === 'income' ? 'text-success-ink' : 'text-error-ink'}">
											{transaksi.type === 'income' ? '+' : '-'}{formatRupiah(transaksi.amount)}
										</span>
									</div>
								</button>
							</li>
						{/each}
					</ul>
				</section>
			{/each}

			{#if kursor}
				<button type="button" class="btn btn-outline min-h-11 w-full" onclick={muatLagi} disabled={memuatLagi}>
					{memuatLagi ? 'Memuat...' : 'Muat lebih banyak'}
				</button>
			{/if}
		</div>
	{/if}
</div>

<Sheet
	open={editor !== null}
	title="Ubah catatan"
	description={editor ? `Dibuat ${labelHari(editor.occurred_at)} ${labelJam(editor.occurred_at)}` : ''}
	onclose={() => (editor = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void simpanEdit();
		}}
	>
		<div class="grid grid-cols-2 gap-2">
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={form.type} value="expense" />
				<span class="text-sm font-medium">Pengeluaran</span>
			</label>
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={form.type} value="income" />
				<span class="text-sm font-medium">Pemasukan</span>
			</label>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="edit-nominal">Nominal</label>
			<MoneyInput id="edit-nominal" value={form.amount} onchange={(value) => (form.amount = value)} />
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="edit-tanggal">Tanggal catatan</label>
			<input id="edit-tanggal" type="date" class="input input-bordered min-h-11 w-full" bind:value={form.tanggal} required />
			<div class="mt-2 flex flex-wrap gap-2">
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (form.tanggal = tanggalInput())}>
					Hari ini
				</button>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (form.tanggal = tanggalKemarin())}>
					Kemarin
				</button>
			</div>
			<p class="mt-2 text-xs opacity-70">
				Jamnya tetap seperti semula, yang berubah hanya tanggalnya.
			</p>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="edit-catatan">Catatan</label>
			<input id="edit-catatan" class="input input-bordered min-h-11 w-full" bind:value={form.note} />
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<label class="label pb-1 text-sm font-semibold" for="edit-kategori">Kategori</label>
				<select id="edit-kategori" class="select select-bordered min-h-11 w-full" bind:value={form.category_id}>
					<option value={0}>Tanpa kategori</option>
					{#each kategori as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="label pb-1 text-sm font-semibold" for="edit-akun">Akun</label>
				<select id="edit-akun" class="select select-bordered min-h-11 w-full" bind:value={form.account_id}>
					<option value={0}>Tanpa akun</option>
					{#each akun as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex flex-wrap items-center justify-between gap-2">
			<button
				type="button"
				class="btn btn-ghost min-h-11 text-error-ink"
				disabled={simpanProses}
				onclick={() => editor && void hapus(editor)}
			>
				Hapus
			</button>
			<div class="flex gap-2">
				<button type="button" class="btn btn-ghost min-h-11" onclick={() => (editor = null)}>Batal</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={simpanProses || form.amount <= 0}>
					{simpanProses ? 'Menyimpan...' : 'Simpan'}
				</button>
			</div>
		</div>
	</form>
</Sheet>

<Sheet open={tambahTerbuka} title="Catat manual" description="Buat catatan tanpa lewat obrolan." onclose={() => (tambahTerbuka = false)}>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void tambah();
		}}
	>
		<div class="grid grid-cols-2 gap-2">
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={formBaru.type} value="expense" />
				<span class="text-sm font-medium">Pengeluaran</span>
			</label>
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={formBaru.type} value="income" />
				<span class="text-sm font-medium">Pemasukan</span>
			</label>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-nominal">Nominal</label>
			<MoneyInput id="baru-nominal" value={formBaru.amount} onchange={(value) => (formBaru.amount = value)} required />
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-tanggal">Tanggal catatan</label>
			<input id="baru-tanggal" type="date" class="input input-bordered min-h-11 w-full" bind:value={formBaru.tanggal} required />
			<div class="mt-2 flex flex-wrap gap-2">
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (formBaru.tanggal = tanggalInput())}>
					Hari ini
				</button>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (formBaru.tanggal = tanggalKemarin())}>
					Kemarin
				</button>
			</div>
			<p class="mt-2 text-xs opacity-70">
				Lupa nyatat kemarin? Pilih tanggalnya di sini, catatannya masuk ke hari itu.
			</p>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-catatan">Catatan</label>
			<input id="baru-catatan" class="input input-bordered min-h-11 w-full" placeholder="mis. kopi" bind:value={formBaru.note} />
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<label class="label pb-1 text-sm font-semibold" for="baru-kategori">Kategori</label>
				<select id="baru-kategori" class="select select-bordered min-h-11 w-full" bind:value={formBaru.category_id}>
					<option value={0}>Tebak dari catatan</option>
					{#each kategori as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="label pb-1 text-sm font-semibold" for="baru-akun">Akun</label>
				<select id="baru-akun" class="select select-bordered min-h-11 w-full" bind:value={formBaru.account_id}>
					<option value={0}>Tanpa akun</option>
					{#each akun as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (tambahTerbuka = false)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={simpanProses || formBaru.amount <= 0}>
				{simpanProses ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
