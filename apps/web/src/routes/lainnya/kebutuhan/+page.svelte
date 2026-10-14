<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { bulanIni, formatRupiah, geserBulan, labelBulan, labelTanggalPendek } from '#lib/format';
	import { toast } from '#lib/toast.svelte';
	import type { Category, Need, RecurringNeed } from '#lib/types';

	let bulan = $state(bulanIni());
	let template = $state<RecurringNeed[]>([]);
	let kebutuhan = $state<Need[]>([]);
	let kategori = $state<Category[]>([]);
	let memuat = $state(true);
	let pesanError = $state('');
	let memproses = $state(false);

	let editor = $state<RecurringNeed | null>(null);
	let tambahTerbuka = $state(false);
	let form = $state({ name: '', amount: 0, due_day: 1, category_id: 0 });

	async function muat() {
		memuat = true;
		pesanError = '';
		try {
			const [templateHasil, kebutuhanHasil] = await Promise.all([
				api.recurringNeeds(),
				api.needs(bulan)
			]);
			template = templateHasil.recurring_needs;
			kebutuhan = kebutuhanHasil.needs;
		} catch (error) {
			pesanError = error instanceof ApiError ? error.message : 'Gagal memuat kebutuhan bulanan.';
		} finally {
			memuat = false;
		}
	}

	onMount(async () => {
		try {
			kategori = (await api.categories()).categories;
		} catch {
			toast('Gagal memuat daftar kategori.', 'error');
		}
		void muat();
	});

	$effect(() => {
		bulan;
		void muat();
	});

	async function generate() {
		memproses = true;
		try {
			const hasil = await api.generateNeeds(bulan);
			toast(hasil.needs > 0 ? `${hasil.needs} kebutuhan disiapkan untuk ${labelBulan(bulan)}.` : 'Semua kebutuhan bulan ini sudah ada.');
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyiapkan kebutuhan.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function simpanTemplate() {
		memproses = true;
		try {
			const payload = {
				name: form.name,
				amount: form.amount,
				due_day: form.due_day,
				category_id: form.category_id || null
			};
			if (editor) {
				await api.updateRecurringNeed(editor.id, payload);
				toast('Kebutuhan diperbarui.');
			} else {
				await api.createRecurringNeed(payload);
				toast('Kebutuhan baru disimpan.');
			}
			editor = null;
			tambahTerbuka = false;
			form = { name: '', amount: 0, due_day: 1, category_id: 0 };
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan kebutuhan.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function nonaktifkan(item: RecurringNeed) {
		memproses = true;
		try {
			await api.updateRecurringNeed(item.id, { active: !item.active });
			toast(item.active ? `${item.name} dimatikan.` : `${item.name} diaktifkan lagi.`);
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mengubah kebutuhan.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function hapus(item: RecurringNeed) {
		memproses = true;
		try {
			await api.deleteRecurringNeed(item.id);
			editor = null;
			toast(`${item.name} dihapus dari template.`);
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus kebutuhan.', 'error');
		} finally {
			memproses = false;
		}
	}

	async function ubahStatus(item: Need) {
		memproses = true;
		try {
			if (item.status === 'paid') {
				await api.unpayNeed(item.id);
				toast(`${item.name} dibuka lagi. Transaksi yang sudah dibuat tetap ada.`);
			} else {
				await api.payNeed(item.id, item.amount);
				toast(`${item.name} ditandai lunas.`);
			}
			await muat();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mengubah status.', 'error');
		} finally {
			memproses = false;
		}
	}
</script>

<svelte:head>
	<title>Kebutuhan bulanan — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Kebutuhan bulanan</h1>
		<button
			type="button"
			class="btn btn-primary btn-sm min-h-11"
			onclick={() => {
				editor = null;
				form = { name: '', amount: 0, due_day: 1, category_id: 0 };
				tambahTerbuka = true;
			}}
		>
			Tambah
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{labelBulan(bulan)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (bulan = geserBulan(bulan, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if memuat}
		<StateBlock state="loading" />
	{:else if pesanError}
		<StateBlock state="error" message={pesanError} actionLabel="Coba lagi" onaction={() => void muat()} />
	{:else}
		<section aria-labelledby="judul-bulan-ini">
			<div class="mb-2 flex items-baseline justify-between gap-2">
				<h2 id="judul-bulan-ini" class="font-display text-base font-semibold">Daftar {labelBulan(bulan)}</h2>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={generate} disabled={memproses}>
					Siapkan dari template
				</button>
			</div>

			{#if kebutuhan.length === 0}
				<StateBlock
					state="empty"
					title="Belum ada kebutuhan bulan ini"
					message="Tambahkan template dulu (mis. listrik tanggal 5), lalu ketuk Siapkan dari template."
				/>
			{:else}
				<ul class="space-y-2">
					{#each kebutuhan as item (item.id)}
						<li class="card bg-base-200">
							<div class="card-body flex-row items-center justify-between gap-3 p-4">
								<div>
									<p class="font-medium {item.status === 'paid' ? 'line-through opacity-70' : ''}">{item.name}</p>
									<p class="text-xs {item.overdue ? 'font-semibold text-error-ink' : 'opacity-70'}">
										{formatRupiah(item.amount)} · jatuh tempo {labelTanggalPendek(item.due_date)}
										{#if item.status === 'paid'}· lunas{/if}
										{#if item.overdue}· telat{/if}
									</p>
								</div>
								<button type="button" class="btn btn-sm min-h-11 {item.status === 'paid' ? 'btn-ghost' : 'btn-primary'}" onclick={() => void ubahStatus(item)} disabled={memproses}>
									{item.status === 'paid' ? 'Batal lunas' : 'Tandai lunas'}
								</button>
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section aria-labelledby="judul-template">
			<h2 id="judul-template" class="mb-2 font-display text-base font-semibold">Template rutin</h2>

			{#if template.length === 0}
				<StateBlock state="empty" title="Belum ada template" message="Template dipakai untuk menyiapkan daftar kebutuhan tiap awal bulan." />
			{:else}
				<ul class="space-y-2">
					{#each template as item (item.id)}
						<li class="card bg-base-200">
							<div class="card-body gap-2 p-4">
								<div class="flex items-baseline justify-between gap-3">
									<span class="font-medium">
										{item.name}
										{#if !item.active}
											<span class="badge badge-ghost badge-sm ml-1">nonaktif</span>
										{/if}
									</span>
									<span class="font-display text-base font-semibold tabular-nums">{formatRupiah(item.amount)}</span>
								</div>
								<p class="text-xs opacity-70">
									Jatuh tempo tanggal {item.due_day}
									{#if item.category_name}· {item.category_name}{/if}
								</p>
								<div class="flex flex-wrap gap-2">
									<button
										type="button"
										class="btn btn-ghost btn-sm min-h-11"
										onclick={() => {
											editor = item;
											form = {
												name: item.name,
												amount: item.amount,
												due_day: item.due_day,
												category_id: item.category_id ?? 0
											};
										}}
									>
										Ubah
									</button>
									<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => void nonaktifkan(item)} disabled={memproses}>
										{item.active ? 'Matikan' : 'Aktifkan'}
									</button>
								</div>
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>

<Sheet
	open={tambahTerbuka || editor !== null}
	title={editor ? `Ubah ${editor.name}` : 'Kebutuhan baru'}
	description="Template ini dipakai untuk menyiapkan daftar kebutuhan tiap bulan."
	onclose={() => {
		tambahTerbuka = false;
		editor = null;
	}}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void simpanTemplate();
		}}
	>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="nama-kebutuhan">Nama</label>
			<input id="nama-kebutuhan" class="input input-bordered min-h-11 w-full" placeholder="mis. listrik" bind:value={form.name} required />
		</div>
		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<label class="label pb-1 text-sm font-semibold" for="nominal-kebutuhan">Perkiraan nominal</label>
				<MoneyInput id="nominal-kebutuhan" value={form.amount} onchange={(value) => (form.amount = value)} />
			</div>
			<div>
				<label class="label pb-1 text-sm font-semibold" for="tanggal-jatuh-tempo">Tanggal jatuh tempo</label>
				<input id="tanggal-jatuh-tempo" type="number" min="1" max="31" class="input input-bordered min-h-11 w-full" bind:value={form.due_day} />
			</div>
		</div>
		<div>
			<label class="label pb-1 text-sm font-semibold" for="kategori-kebutuhan">Kategori</label>
			<select id="kategori-kebutuhan" class="select select-bordered min-h-11 w-full" bind:value={form.category_id}>
				<option value={0}>Tanpa kategori</option>
				{#each kategori as item (item.id)}
					<option value={item.id}>{item.name}</option>
				{/each}
			</select>
		</div>
		<div class="flex flex-wrap items-center justify-between gap-2">
			{#if editor}
				<button type="button" class="btn btn-ghost min-h-11 text-error-ink" onclick={() => editor && void hapus(editor)} disabled={memproses}>
					Hapus
				</button>
			{/if}
			<div class="flex gap-2">
				<button
					type="button"
					class="btn btn-ghost min-h-11"
					onclick={() => {
						tambahTerbuka = false;
						editor = null;
					}}
				>
					Batal
				</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={memproses || !form.name.trim()}>
					{memproses ? 'Menyimpan...' : 'Simpan'}
				</button>
			</div>
		</div>
	</form>
</Sheet>
