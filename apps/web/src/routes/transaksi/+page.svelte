<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { onChatChange } from '#lib/chat.svelte';
	import { hapticLight } from '#lib/haptic';
	import {
		monthFromDateInput,
		currentMonth,
		formatRupiah,
		shiftMonth,
		isoFromDateInput,
		formatMonth,
		formatDay,
		formatTime,
		toDateInputValue,
		yesterdayInputValue
	} from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Account, Category, Transaction } from '#lib/types';

	let month = $state(currentMonth());
	let transactions = $state<Transaction[]>([]);
	let categories = $state<Category[]>([]);
	let accounts = $state<Account[]>([]);
	let cursor = $state<number | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let errorMessage = $state('');

	let searchQuery = $state('');
	let categoryFilter = $state<number | 0>(0);

	let editor = $state<Transaction | null>(null);
	let form = $state({
		type: 'expense',
		amount: 0,
		note: '',
		category_id: 0,
		account_id: 0,
		date: toDateInputValue()
	});
	let saving = $state(false);

	let addSheetOpen = $state(false);
	let newForm = $state({
		type: 'expense',
		amount: 0,
		note: '',
		category_id: 0,
		account_id: 0,
		date: toDateInputValue()
	});

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			const result = await api.transactions({
				month,
				q: searchQuery.trim() || undefined,
				category_id: categoryFilter || undefined,
				limit: 60
			});
			transactions = result.transactions;
			cursor = result.next_cursor;
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat transaksi.';
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (!cursor || loadingMore) return;
		loadingMore = true;
		try {
			const result = await api.transactions({
				month,
				q: searchQuery.trim() || undefined,
				category_id: categoryFilter || undefined,
				cursor,
				limit: 60
			});
			transactions = [...transactions, ...result.transactions];
			cursor = result.next_cursor;
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal memuat lanjutan.', 'error');
		} finally {
			loadingMore = false;
		}
	}

	onMount(() => {
		void (async () => {
			try {
				const [categoriesResult, accountsResult] = await Promise.all([api.categories(), api.accounts()]);
				categories = categoriesResult.categories;
				accounts = accountsResult.accounts;
			} catch {
				toast('Gagal memuat daftar kategori dan akun.', 'error');
			}
		})();

		return onChatChange(() => void load());
	});

	$effect(() => {
		month;
		categoryFilter;
		void load();
	});

	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	function onSearchInput(value: string) {
		searchQuery = value;
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => void load(), 350);
	}

	const groups = $derived.by(() => {
		const result: { title: string; items: Transaction[] }[] = [];
		for (const item of transactions) {
			const title = formatDay(item.occurred_at);
			const last = result[result.length - 1];
			if (last && last.title === title) {
				last.items.push(item);
			} else {
				result.push({ title, items: [item] });
			}
		}
		return result;
	});

	function openEditor(transaction: Transaction) {
		editor = transaction;
		form = {
			type: transaction.type,
			amount: transaction.amount,
			note: transaction.note,
			category_id: transaction.category_id ?? 0,
			account_id: transaction.account_id ?? 0,
			date: toDateInputValue(transaction.occurred_at)
		};
	}

	// A record dated into another month leaves the open list, so tell the user where it landed.
	function notifyMonthChange(date: string) {
		const targetMonth = monthFromDateInput(date);
		if (targetMonth !== month) {
			toast(`Tersimpan di ${formatMonth(targetMonth)}.`);
			return true;
		}
		return false;
	}

	async function saveEdit() {
		if (!editor) return;
		saving = true;
		try {
			await api.updateTransaction(editor.id, {
				type: form.type,
				amount: form.amount,
				note: form.note,
				category_id: form.category_id || null,
				account_id: form.account_id || null,
				clear_category: form.category_id === 0,
				clear_account: form.account_id === 0,
				// Only the date changes; the original time of day is preserved.
				occurred_at: isoFromDateInput(form.date, editor.occurred_at)
			});
			const monthChanged = notifyMonthChange(form.date);
			editor = null;
			hapticLight();
			if (!monthChanged) toast('Perubahan tersimpan.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan perubahan.', 'error');
		} finally {
			saving = false;
		}
	}

	async function remove(transaction: Transaction) {
		try {
			await api.deleteTransaction(transaction.id);
			editor = null;
			toast('Transaksi dihapus.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus transaksi.', 'error');
		}
	}

	// While viewing another month, new entries default there instead of silently using today.
	function defaultDate() {
		return month === currentMonth() ? toDateInputValue() : `${month}-01`;
	}

	function openAdd() {
		newForm = {
			type: 'expense',
			amount: 0,
			note: '',
			category_id: 0,
			account_id: 0,
			date: defaultDate()
		};
		addSheetOpen = true;
	}

	async function createTransaction() {
		saving = true;
		try {
			await api.createTransaction({
				type: newForm.type,
				amount: newForm.amount,
				note: newForm.note,
				category_id: newForm.category_id || null,
				account_id: newForm.account_id || null,
				occurred_at: isoFromDateInput(newForm.date)
			});
			const monthChanged = notifyMonthChange(newForm.date);
			addSheetOpen = false;
			newForm = {
				type: 'expense',
				amount: 0,
				note: '',
				category_id: 0,
				account_id: 0,
				date: defaultDate()
			};
			hapticLight();
			if (!monthChanged) toast('Catatan baru tersimpan.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan catatan.', 'error');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Transaksi — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Transaksi</h1>
		<button type="button" class="btn btn-primary btn-sm min-h-11" onclick={openAdd}>
			Tambah manual
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, -1))} aria-label="Bulan sebelumnya">
			‹
		</button>
		<span class="font-display text-base font-semibold">{formatMonth(month)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, 1))} aria-label="Bulan berikutnya">
			›
		</button>
	</div>

	<div class="flex flex-col gap-2 sm:flex-row">
		<div class="flex-1">
			<label class="sr-only" for="transaction-search">Cari catatan</label>
			<input
				id="transaction-search"
				type="search"
				class="input input-bordered min-h-11 w-full"
				placeholder="Cari catatan, mis. kopi"
				value={searchQuery}
				oninput={(event) => onSearchInput(event.currentTarget.value)}
			/>
		</div>
		<div>
			<label class="sr-only" for="category-filter">Filter kategori</label>
			<select id="category-filter" class="select select-bordered min-h-11 w-full sm:w-44" bind:value={categoryFilter}>
				<option value={0}>Semua kategori</option>
				{#each categories as item (item.id)}
					<option value={item.id}>{item.name}</option>
				{/each}
			</select>
		</div>
	</div>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else if transactions.length === 0}
		<StateBlock
			state="empty"
			title="Belum ada transaksi di bulan ini"
			message="Coba ganti bulan, ubah filter, atau catat lewat obrolan di Home."
			actionLabel="Tambah manual"
			onaction={openAdd}
		/>
	{:else}
		<div class="space-y-4">
			{#each groups as group (group.title)}
				<section aria-label={group.title}>
					<h2 class="mb-2 text-xs font-semibold tracking-wide uppercase opacity-70">{group.title}</h2>
					<ul class="anim-list space-y-2">
						{#each group.items as transaction (transaction.id)}
							<li class="card bg-base-200">
								<button
									type="button"
									class="w-full rounded-box text-left transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
									onclick={() => openEditor(transaction)}
								>
									<div class="card-body flex-row items-center justify-between gap-3 p-4">
										<div class="min-w-0">
											<p class="truncate font-medium">{transaction.note || 'Tanpa catatan'}</p>
											<p class="text-xs opacity-70">
												{formatTime(transaction.occurred_at)}
												{#if transaction.category_name}· {transaction.category_name}{/if}
												{#if transaction.account_name}· {transaction.account_name}{/if}
												{#if transaction.source === 'telegram'}· dari Telegram{/if}
												{#if transaction.source === 'need'}· dari kebutuhan{/if}
											</p>
										</div>
										<span class="font-display text-base font-semibold tabular-nums {transaction.type === 'income' ? 'text-success-ink' : 'text-error-ink'}">
											{transaction.type === 'income' ? '+' : '-'}{formatRupiah(transaction.amount)}
										</span>
									</div>
								</button>
							</li>
						{/each}
					</ul>
				</section>
			{/each}

			{#if cursor}
				<button type="button" class="btn btn-outline min-h-11 w-full" onclick={loadMore} disabled={loadingMore}>
					{loadingMore ? 'Memuat...' : 'Muat lebih banyak'}
				</button>
			{/if}
		</div>
	{/if}
</div>

<Sheet
	open={editor !== null}
	title="Ubah catatan"
	description={editor ? `Dibuat ${formatDay(editor.occurred_at)} ${formatTime(editor.occurred_at)}` : ''}
	onclose={() => (editor = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void saveEdit();
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
			<label class="label pb-1 text-sm font-semibold" for="edit-date">Tanggal catatan</label>
			<input id="edit-date" type="date" class="input input-bordered min-h-11 w-full" bind:value={form.date} required />
			<div class="mt-2 flex flex-wrap gap-2">
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (form.date = toDateInputValue())}>
					Hari ini
				</button>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (form.date = yesterdayInputValue())}>
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
					{#each categories as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="label pb-1 text-sm font-semibold" for="edit-akun">Akun</label>
				<select id="edit-akun" class="select select-bordered min-h-11 w-full" bind:value={form.account_id}>
					<option value={0}>Tanpa akun</option>
					{#each accounts as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex flex-wrap items-center justify-between gap-2">
			<button
				type="button"
				class="btn btn-ghost min-h-11 text-error-ink"
				disabled={saving}
				onclick={() => editor && void remove(editor)}
			>
				Hapus
			</button>
			<div class="flex gap-2">
				<button type="button" class="btn btn-ghost min-h-11" onclick={() => (editor = null)}>Batal</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={saving || form.amount <= 0}>
					{saving ? 'Menyimpan...' : 'Simpan'}
				</button>
			</div>
		</div>
	</form>
</Sheet>

<Sheet open={addSheetOpen} title="Catat manual" description="Buat catatan tanpa lewat obrolan." onclose={() => (addSheetOpen = false)}>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void createTransaction();
		}}
	>
		<div class="grid grid-cols-2 gap-2">
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={newForm.type} value="expense" />
				<span class="text-sm font-medium">Pengeluaran</span>
			</label>
			<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-base-300 px-3">
				<input type="radio" class="radio" bind:group={newForm.type} value="income" />
				<span class="text-sm font-medium">Pemasukan</span>
			</label>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-nominal">Nominal</label>
			<MoneyInput id="baru-nominal" value={newForm.amount} onchange={(value) => (newForm.amount = value)} required />
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-tanggal">Tanggal catatan</label>
			<input id="baru-tanggal" type="date" class="input input-bordered min-h-11 w-full" bind:value={newForm.date} required />
			<div class="mt-2 flex flex-wrap gap-2">
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (newForm.date = toDateInputValue())}>
					Hari ini
				</button>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={() => (newForm.date = yesterdayInputValue())}>
					Kemarin
				</button>
			</div>
			<p class="mt-2 text-xs opacity-70">
				Lupa nyatat kemarin? Pilih tanggalnya di sini, catatannya masuk ke hari itu.
			</p>
		</div>

		<div>
			<label class="label pb-1 text-sm font-semibold" for="baru-catatan">Catatan</label>
			<input id="baru-catatan" class="input input-bordered min-h-11 w-full" placeholder="mis. kopi" bind:value={newForm.note} />
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<label class="label pb-1 text-sm font-semibold" for="baru-kategori">Kategori</label>
				<select id="baru-kategori" class="select select-bordered min-h-11 w-full" bind:value={newForm.category_id}>
					<option value={0}>Tebak dari catatan</option>
					{#each categories as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="label pb-1 text-sm font-semibold" for="baru-akun">Akun</label>
				<select id="baru-akun" class="select select-bordered min-h-11 w-full" bind:value={newForm.account_id}>
					<option value={0}>Tanpa akun</option>
					{#each accounts as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex justify-end gap-2">
			<button type="button" class="btn btn-ghost min-h-11" onclick={() => (addSheetOpen = false)}>Batal</button>
			<button type="submit" class="btn btn-primary min-h-11" disabled={saving || newForm.amount <= 0}>
				{saving ? 'Menyimpan...' : 'Simpan'}
			</button>
		</div>
	</form>
</Sheet>
