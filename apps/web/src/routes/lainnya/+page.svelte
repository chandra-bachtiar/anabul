<script lang="ts">
	import { page } from '$app/state';

	const menu = [
		{
			href: '/lainnya/kebutuhan',
			judul: 'Kebutuhan bulanan',
			ket: 'Listrik, internet, cicilan, dan tagihan rutin lain.'
		},
		{
			href: '/tutup-bulan',
			judul: 'Tutup bulan',
			ket: 'Lihat ringkasan, kunci bulan, lalu siapkan bulan berikutnya.'
		},
		{
			href: '/lainnya/kategori',
			judul: 'Kategori dan akun',
			ket: 'Atur kategori pengeluaran dan daftar akun (cash, bank, e-wallet).'
		},
		{
			href: '/lainnya/pengaturan',
			judul: 'Pengaturan',
			ket: 'Tema terang atau gelap, info versi, dan keluar dari sesi.'
		}
	];

	const jalur = $derived(page.url.pathname);
</script>

<svelte:head>
	<title>Lainnya — ANABUL</title>
</svelte:head>

<div class="space-y-4">
	<h1 class="font-display text-2xl font-semibold">Lainnya</h1>

	<ul class="space-y-2">
		{#each menu as item (item.href)}
			<li>
				<a
					href={item.href}
					aria-current={jalur === item.href ? 'page' : undefined}
					class="card bg-base-200 transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
				>
					<div class="card-body flex-row items-center justify-between gap-3 p-4">
						<div>
							<p class="font-medium">{item.judul}</p>
							<p class="text-xs opacity-70">{item.ket}</p>
						</div>
						<svg viewBox="0 0 24 24" class="h-5 w-5 shrink-0 opacity-70" fill="none" aria-hidden="true">
							<path d="M9.5 6.5 15 12l-5.5 5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
						</svg>
					</div>
				</a>
			</li>
		{/each}
	</ul>

	<p class="px-1 text-xs opacity-70">
		ANABUL dipakai sendiri, jadi semua catatan di sini milikmu. Database bisa diarahkan ke Postgres lokal
		atau Neon, cukup ganti DATABASE_URL.
	</p>
</div>
