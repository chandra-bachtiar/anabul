<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { toast } from '#lib/toast.svelte';
	import type { Account, Category } from '#lib/types';

	let kategori = $state<Category[]>([]);
	let akun = $state<Account[]>([]);
	let memuat = $state(true);
	let pesanError = $state('');
	let memproses = $state(false);

	let kategoriBaru = $state('');
	let akunBaru = $state('');
	let editor = $state<Category | null>(null);
	let formKategori = $state({ name: '', keywords: '' });

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			const [kategoriHasil, akunHasil] = await Promise.all([api.categories(), api.accounts()]);
			kategori = kategoriHasil.categories;
			akun = akunHasil.accounts;
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat kategori dan akun.';
		} finally {
			memuat = false;
		}
	}

	onMount(() => void muat());

	async function tambahKategori() {
		if (!kategoriBaru.trim()) return;
		memproses = true;
		try {
			await api.createCategory(kategoriBaru.trim());
			toast(`Kategori ${kategoriBaru.trim()} ditambahkan.`);
			kategoriBaru = '';
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menambah kategori.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function simpanKategori() {
		if (!editor) return;
		memproses = true;
		try {
			const keywords = formKategori.keywords
				.split(',')
				.map((item) => item.trim())
				.filter((item) => item.length > 0);
			await api.updateCategory(editor.id, { name: formKategori.name, keywords });
			toast('Kategori diperbarui.');
			editor = null;
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan kategori.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function hapusKategori(item: Category) {
		memproses = true;
		try {
			await api.deleteCategory(item.id);
			toast(`Kategori ${item.name} dihapus. Transaksi lamanya jadi tanpa kategori.`);
			editor = null;
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus kategori.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function tambahAkun() {
		if (!akunBaru.trim()) return;
		memproses = true;
		try {
			await api.createAccount(akunBaru.trim());
			toast(`Akun ${akunBaru.trim()} ditambahkan.`);
			akunBaru = '';
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menambah akun.', 'error');
		} finally {
			memproses = false;
		}
	}
</script>

<svelte:head>
	<title>Kategori dan akun — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Kategori dan akun</h1>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else}
		<section class="card bg-base-200" aria-labelledby="judul-kategori">
			<div class="card-body gap-3 p-4">
				<h2 id="judul-kategori" class="font-display text-base font-semibold">Kategori</h2>

				<form
					class="flex gap-2"
					onsubmit={(event) => {
						event.preventDefault();
						void tambahKategori();
					}}
				>
					<label class="sr-only" for="kategori-baru">Nama kategori baru</label>
					<input id="kategori-baru" class="input input-bordered min-h-11 w-full" placeholder="mis. langganan" bind:value={kategoriBaru} />
					<button type="submit" class="btn btn-primary min-h-11" disabled={memproses || !kategoriBaru.trim()}>Tambah</button>
				</form>

				<ul class="space-y-2">
					{#each kategori as item (item.id)}
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
									formKategori = { name: item.name, keywords: item.keywords.join(', ') };
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
						void tambahAkun();
					}}
				>
					<label class="sr-only" for="akun-baru">Nama akun baru</label>
					<input id="akun-baru" class="input input-bordered min-h-11 w-full" placeholder="mis. dompet digital" bind:value={akunBaru} />
					<button type="submit" class="btn btn-primary min-h-11" disabled={memproses || !akunBaru.trim()}>Tambah</button>
				</form>

				<ul class="flex flex-wrap gap-2">
					{#each akun as item (item.id)}
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
			void simpanKategori();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nama-kategori">Nama</label>
			<input id="nama-kategori" class="input input-bordered min-h-11 w-full" bind:value={formKategori.name} required />
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="kata-kunci">Kata kunci</label>
			<input id="kata-kunci" class="input input-bordered min-h-11 w-full" placeholder="kopi, jajan, warung" bind:value={formKategori.keywords} />
		</div>
		<div class="flex flex-wrap items-center justify-between gap-2">
			{#if editor}
				<button type="button" class="btn btn-ghost min-h-11 text-error-ink" onclick={() => editor && void hapusKategori(editor)} disabled={memproses}>
					Hapus kategori
				</button>
			{/if}
			<div class="flex gap-2">
				<button type="button" class="btn btn-ghost min-h-11" onclick={() => (editor = null)}>Batal</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={memproses}>Simpan</button>
			</div>
		</div>
	</form>
</Sheet>
