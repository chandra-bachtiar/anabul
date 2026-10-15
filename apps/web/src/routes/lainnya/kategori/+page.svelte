<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { toast } from '#lib/toast.svelte';
	import type { Account, Category } from '#lib/types';

	let categories = $state<Category[]>([]);
	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let processing = $state(false);

	let newCategory = $state('');
	let newAccount = $state('');
	let editor = $state<Category | null>(null);
	let categoryForm = $state({ name: '', keywords: '' });

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			const [categoriesResult, accountsResult] = await Promise.all([api.categories(), api.accounts()]);
			categories = categoriesResult.categories;
			accounts = accountsResult.accounts;
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat kategori dan akun.';
		} finally {
			loading = false;
		}
	}

	onMount(() => void load());

	async function createCategory() {
		if (!newCategory.trim()) return;
		processing = true;
		try {
			await api.createCategory(newCategory.trim());
			toast(`Kategori ${newCategory.trim()} ditambahkan.`);
			newCategory = '';
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menambah kategori.', 'error');
		} finally {
			processing = false;
		}
	}

	async function saveCategory() {
		if (!editor) return;
		processing = true;
		try {
			const keywords = categoryForm.keywords
				.split(',')
				.map((item) => item.trim())
				.filter((item) => item.length > 0);
			await api.updateCategory(editor.id, { name: categoryForm.name, keywords });
			toast('Kategori diperbarui.');
			editor = null;
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan kategori.', 'error');
		} finally {
			processing = false;
		}
	}

	async function removeCategory(item: Category) {
		processing = true;
		try {
			await api.deleteCategory(item.id);
			toast(`Kategori ${item.name} dihapus. Transaksi lamanya jadi tanpa kategori.`);
			editor = null;
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus kategori.', 'error');
		} finally {
			processing = false;
		}
	}

	async function createAccount() {
		if (!newAccount.trim()) return;
		processing = true;
		try {
			await api.createAccount(newAccount.trim());
			toast(`Akun ${newAccount.trim()} ditambahkan.`);
			newAccount = '';
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menambah akun.', 'error');
		} finally {
			processing = false;
		}
	}
</script>

<svelte:head>
	<title>Kategori dan akun — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Kategori dan akun</h1>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else}
		<section class="card bg-base-200" aria-labelledby="category-title">
			<div class="card-body gap-3 p-4">
				<h2 id="category-title" class="font-display text-base font-semibold">Kategori</h2>

				<form
					class="flex gap-2"
					onsubmit={(event) => {
						event.preventDefault();
						void createCategory();
					}}
				>
					<label class="sr-only" for="kategori-baru">Nama kategori baru</label>
					<input id="kategori-baru" class="input input-bordered min-h-11 w-full" placeholder="mis. langganan" bind:value={newCategory} />
					<button type="submit" class="btn btn-primary min-h-11" disabled={processing || !newCategory.trim()}>Tambah</button>
				</form>

				<ul class="space-y-2">
					{#each categories as item (item.id)}
						<li class="flex items-center justify-between gap-2 rounded-xl bg-base-300/40 px-3 py-2">
							<div class="min-w-0">
								<p class="truncate text-sm font-medium">{item.name}</p>
								{#if item.keywords.length > 0}
									<p class="truncate text-xs opacity-70">kata kunci: {item.keywords.join(', ')}</p>
								{/if}
							</div>
							<button
								type="button"
								class="btn btn-ghost btn-sm min-h-11"
								onclick={() => {
									editor = item;
									categoryForm = { name: item.name, keywords: item.keywords.join(', ') };
								}}
							>
								Ubah
							</button>
						</li>
					{/each}
				</ul>

				<p class="text-xs opacity-70">
					Kata kunci dipakai untuk menebak kategori otomatis dari catatan, mis. "kopi" masuk ke
					makan-minum.
				</p>
			</div>
		</section>

		<section class="card bg-base-200" aria-labelledby="judul-akun">
			<div class="card-body gap-3 p-4">
				<h2 id="judul-akun" class="font-display text-base font-semibold">Akun</h2>

				<form
					class="flex gap-2"
					onsubmit={(event) => {
						event.preventDefault();
						void createAccount();
					}}
				>
					<label class="sr-only" for="akun-baru">Nama akun baru</label>
					<input id="akun-baru" class="input input-bordered min-h-11 w-full" placeholder="mis. dompet digital" bind:value={newAccount} />
					<button type="submit" class="btn btn-primary min-h-11" disabled={processing || !newAccount.trim()}>Tambah</button>
				</form>

				<ul class="flex flex-wrap gap-2">
					{#each accounts as item (item.id)}
						<li class="badge badge-lg bg-base-300/60 px-3 py-3">{item.name}</li>
					{/each}
				</ul>

				<p class="text-xs opacity-70">Akun dipakai kalau kamu mau memisahkan cash, bank, dan e-wallet.</p>
			</div>
		</section>
	{/if}
</div>

<Sheet
	open={editor !== null}
	title={editor ? `Ubah ${editor.name}` : 'Ubah kategori'}
	description="Kata kunci dipisah dengan koma."
	onclose={() => (editor = null)}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void saveCategory();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nama-kategori">Nama</label>
			<input id="nama-kategori" class="input input-bordered min-h-11 w-full" bind:value={categoryForm.name} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="kata-kunci">Kata kunci</label>
			<input id="kata-kunci" class="input input-bordered min-h-11 w-full" placeholder="kopi, jajan, warung" bind:value={categoryForm.keywords} />
		</div>
		<div class="flex flex-wrap items-center justify-between gap-2">
			{#if editor}
				<button type="button" class="btn btn-ghost min-h-11 text-error-ink" onclick={() => editor && void removeCategory(editor)} disabled={processing}>
					Hapus kategori
				</button>
			{/if}
			<div class="flex gap-2">
				<button type="button" class="btn btn-ghost min-h-11" onclick={() => (editor = null)}>Batal</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={processing}>Simpan</button>
			</div>
		</div>
	</form>
</Sheet>
