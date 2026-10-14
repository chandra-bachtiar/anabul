<script lang="ts">
	import {
		batalAksi,
		batalkanTransaksi,
		chatState,
		konfirmasiAksi,
		kirimPesan,
		ulangiPesanTerakhir,
		type PesanChat
	} from '#lib/chat.svelte';

	let teks = $state('');
	let daftar = $state<HTMLDivElement | null>(null);

	function kirim(event: SubmitEvent) {
		event.preventDefault();
		const isi = teks;
		teks = '';
		void kirimPesan(isi);
	}

	// Gulir otomatis ke pesan terbaru supaya balasan agent selalu terlihat.
	$effect(() => {
		const jumlah = chatState.items.length;
		const node = daftar;
		if (!node || jumlah === 0) return;
		node.scrollTop = node.scrollHeight;
	});
</script>

<section class="card bg-base-200" aria-labelledby="judul-chat">
	<div class="card-body gap-3 p-4">
		<div class="flex items-baseline justify-between gap-2">
			<h2 id="judul-chat" class="font-display text-base font-semibold">Ngobrol sama ANABUL</h2>
			<span class="text-xs opacity-70">contoh: "kopi 25rb"</span>
		</div>

		{#if chatState.items.length > 0}
			<div
				bind:this={daftar}
				class="max-h-72 space-y-2 overflow-y-auto pr-1"
				role="log"
				aria-label="Percakapan dengan agent"
			>
				{#each chatState.items as pesan (pesan.id)}
					{#if pesan.role === 'user'}
						<p class="ml-auto max-w-[85%] w-fit rounded-2xl rounded-br-md bg-primary px-3 py-2 text-sm font-medium text-primary-content">
							{pesan.text}
						</p>
					{:else}
						<div class="max-w-[92%] space-y-2">
							<p
								class="w-fit rounded-2xl rounded-bl-md px-3 py-2 text-sm whitespace-pre-line {pesan.gagal
									? 'bg-error/15 text-error-ink'
									: 'bg-base-100'}"
							>
								{pesan.text}
							</p>

							{#if pesan.pending && !pesan.pending.selesai}
								<div class="card border border-warning/50 bg-warning/10">
									<div class="card-body gap-2 p-3">
										<p class="text-xs font-semibold tracking-wide text-warning-content uppercase">Butuh konfirmasi</p>
										<p class="text-sm">{pesan.pending.summary}</p>
										<div class="flex flex-wrap gap-2">
											<button
												type="button"
												class="btn btn-warning btn-sm min-h-11"
												disabled={chatState.mengirim}
												onclick={() => void konfirmasiAksi(pesan)}
											>
												Ya, lanjutkan
											</button>
											<button
												type="button"
												class="btn btn-ghost btn-sm min-h-11"
												disabled={chatState.mengirim}
												onclick={() => void batalAksi(pesan)}
											>
												Batal
											</button>
										</div>
									</div>
								</div>
							{/if}

							{#if pesan.undo}
								<button
									type="button"
									class="btn btn-outline btn-sm min-h-11"
									disabled={pesan.undo.dipakai}
									onclick={() => void batalkanTransaksi(pesan)}
								>
									{pesan.undo.dipakai ? 'Sudah dibatalkan' : 'Undo catatan ini'}
								</button>
							{/if}

							{#if pesan.gagal}
								<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => void ulangiPesanTerakhir()}>
									Coba kirim lagi
								</button>
							{/if}
						</div>
					{/if}
				{/each}

				{#if chatState.mengirim}
					<p class="w-fit rounded-2xl rounded-bl-md bg-base-100 px-3 py-2 text-sm opacity-70" aria-live="polite">
						ANABUL lagi mikir...
					</p>
				{/if}
			</div>
		{/if}

		<form class="flex items-end gap-2" onsubmit={kirim}>
			<label class="sr-only" for="pesan-chat">Tulis catatan atau pertanyaan</label>
			<textarea
				id="pesan-chat"
				bind:value={teks}
				rows="1"
				placeholder={'ketik aja, mis. "kopi 25rb"'}
				class="textarea textarea-bordered min-h-11 w-full resize-none text-base leading-snug"
				onkeydown={(event) => {
					if (event.key === 'Enter' && !event.shiftKey) {
						event.preventDefault();
						const isi = teks;
						teks = '';
						void kirimPesan(isi);
					}
				}}
			></textarea>
			<button
				type="submit"
				class="btn btn-primary min-h-11 px-4"
				disabled={chatState.mengirim || teks.trim().length === 0}
			>
				Kirim
			</button>
		</form>

		<p class="text-xs opacity-70">
			ANABUL mencatat lewat tool yang divalidasi. Aksi berisiko minta konfirmasi dulu.
		</p>
	</div>
</section>
