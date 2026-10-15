<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, api } from '#lib/api';
	import Confetti from '#lib/components/Confetti.svelte';
	import MoneyInput from '#lib/components/MoneyInput.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import StateBlock from '#lib/components/StateBlock.svelte';
	import { currentMonth, formatRupiah, shiftMonth, formatMonth, formatDateShort } from '#lib/format';
	import { hapticLight } from '#lib/haptic';
	import { toast } from '#lib/toast.svelte';
	import type { Category, Need, RecurringNeed } from '#lib/types';

	let month = $state(currentMonth());
	let templates = $state<RecurringNeed[]>([]);
	let needs = $state<Need[]>([]);
	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let processing = $state(false);
	let celebration = $state(0);

	let editor = $state<RecurringNeed | null>(null);
	let addSheetOpen = $state(false);
	let form = $state({ name: '', amount: 0, due_day: 1, category_id: 0 });

	async function load() {
		loading = true;
		errorMessage = '';
		try {
			const [templatesResult, needsResult] = await Promise.all([
				api.recurringNeeds(),
				api.needs(month)
			]);
			templates = templatesResult.recurring_needs;
			needs = needsResult.needs;
		} catch (error) {
			errorMessage = error instanceof ApiError ? error.message : 'Gagal memuat kebutuhan bulanan.';
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		try {
			categories = (await api.categories()).categories;
		} catch {
			toast('Gagal memuat daftar kategori.', 'error');
		}
		void load();
	});

	$effect(() => {
		month;
		void load();
	});

	async function generateNeeds() {
		processing = true;
		try {
			const result = await api.generateNeeds(month);
			toast(result.needs > 0 ? `${result.needs} kebutuhan disiapkan untuk ${formatMonth(month)}.` : 'Semua kebutuhan bulan ini sudah ada.');
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyiapkan kebutuhan.', 'error');
		} finally {
			processing = false;
		}
	}

	async function saveTemplate() {
		processing = true;
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
			addSheetOpen = false;
			form = { name: '', amount: 0, due_day: 1, category_id: 0 };
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menyimpan kebutuhan.', 'error');
		} finally {
			processing = false;
		}
	}

	async function toggleActive(item: RecurringNeed) {
		processing = true;
		try {
			await api.updateRecurringNeed(item.id, { active: !item.active });
			toast(item.active ? `${item.name} dimatikan.` : `${item.name} diaktifkan lagi.`);
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mengubah kebutuhan.', 'error');
		} finally {
			processing = false;
		}
	}

	async function remove(item: RecurringNeed) {
		processing = true;
		try {
			await api.deleteRecurringNeed(item.id);
			editor = null;
			toast(`${item.name} dihapus dari template.`);
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal menghapus kebutuhan.', 'error');
		} finally {
			processing = false;
		}
	}

	async function toggleStatus(item: Need) {
		processing = true;
		try {
			if (item.status === 'paid') {
				await api.unpayNeed(item.id);
				hapticLight();
				toast(`${item.name} dibuka lagi. Transaksi yang sudah dibuat tetap ada.`);
			} else {
				await api.payNeed(item.id, item.amount);
				celebration += 1;
				hapticLight();
				toast(`${item.name} ditandai lunas.`);
			}
			await load();
		} catch (error) {
			toast(error instanceof ApiError ? error.message : 'Gagal mengubah status.', 'error');
		} finally {
			processing = false;
		}
	}
</script>

<svelte:head>
	<title>Kebutuhan bulanan — ANABUL</title>
</svelte:head>

<Confetti trigger={celebration} />

<div class="space-y-4">
	<header class="flex items-center justify-between gap-2">
		<h1 class="font-display text-2xl font-semibold">Kebutuhan bulanan</h1>
		<button
			type="button"
			class="btn btn-primary btn-sm min-h-11"
			onclick={() => {
				editor = null;
				form = { name: '', amount: 0, due_day: 1, category_id: 0 };
				addSheetOpen = true;
			}}
		>
			Tambah
		</button>
	</header>

	<div class="flex items-center justify-between gap-2">
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, -1))} aria-label="Bulan sebelumnya">‹</button>
		<span class="font-display text-base font-semibold">{formatMonth(month)}</span>
		<button type="button" class="btn btn-ghost min-h-11 min-w-11 px-3" onclick={() => (month = shiftMonth(month, 1))} aria-label="Bulan berikutnya">›</button>
	</div>

	{#if loading}
		<StateBlock state="loading" />
	{:else if errorMessage}
		<StateBlock state="error" message={errorMessage} actionLabel="Coba lagi" onaction={() => void load()} />
	{:else}
		<section aria-labelledby="judul-bulan-ini">
			<div class="mb-2 flex items-baseline justify-between gap-2">
				<h2 id="judul-bulan-ini" class="font-display text-base font-semibold">Daftar {formatMonth(month)}</h2>
				<button type="button" class="btn btn-outline btn-sm min-h-11" onclick={generateNeeds} disabled={processing}>
					Siapkan dari template
				</button>
			</div>

			{#if needs.length === 0}
				<StateBlock
					state="empty"
					title="Belum ada kebutuhan bulan ini"
					message="Tambahkan template dulu (mis. listrik tanggal 5), lalu ketuk Siapkan dari template."
				/>
			{:else}
				<ul class="anim-list space-y-2">
					{#each needs as item (item.id)}
						<li class="card bg-base-200">
							<div class="card-body flex-row items-center justify-between gap-3 p-4">
								<div>
									<p class="font-medium {item.status === 'paid' ? 'line-through opacity-70' : ''}">{item.name}</p>
									<p class="text-xs {item.overdue ? 'font-semibold text-error-ink' : 'opacity-70'}">
										{formatRupiah(item.amount)} · jatuh tempo {formatDateShort(item.due_date)}
										{#if item.status === 'paid'}· lunas{/if}
										{#if item.overdue}· telat{/if}
									</p>
								</div>
								<button type="button" class="btn btn-sm min-h-11 {item.status === 'paid' ? 'btn-ghost' : 'btn-primary'}" onclick={() => void toggleStatus(item)} disabled={processing}>
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

			{#if templates.length === 0}
				<StateBlock state="empty" title="Belum ada template" message="Template dipakai untuk menyiapkan daftar kebutuhan tiap awal bulan." />
			{:else}
				<ul class="anim-list space-y-2">
					{#each templates as item (item.id)}
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
									<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => void toggleActive(item)} disabled={processing}>
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
	open={addSheetOpen || editor !== null}
	title={editor ? `Ubah ${editor.name}` : 'Kebutuhan baru'}
	description="Template ini dipakai untuk menyiapkan daftar kebutuhan tiap bulan."
	onclose={() => {
		addSheetOpen = false;
		editor = null;
	}}
>
	<form
		class="space-y-4"
		onsubmit={(event) => {
			event.preventDefault();
			void saveTemplate();
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
				{#each categories as item (item.id)}
					<option value={item.id}>{item.name}</option>
				{/each}
			</select>
		</div>
		<div class="flex flex-wrap items-center justify-between gap-2">
			{#if editor}
				<button type="button" class="btn btn-ghost min-h-11 text-error-ink" onclick={() => editor && void remove(editor)} disabled={processing}>
					Hapus
				</button>
			{/if}
			<div class="flex gap-2">
				<button
					type="button"
					class="btn btn-ghost min-h-11"
					onclick={() => {
						addSheetOpen = false;
						editor = null;
					}}
				>
					Batal
				</button>
				<button type="submit" class="btn btn-primary min-h-11" disabled={processing || !form.name.trim()}>
					{processing ? 'Menyimpan...' : 'Simpan'}
				</button>
			</div>
		</div>
	</form>
</Sheet>
