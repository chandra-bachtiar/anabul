<script lang="ts">
	import {
		cancelAction,
		chatState,
		confirmAction,
		retryLastMessage,
		sendMessage,
		undoTransaction,
		type ChatMessage
	} from '#lib/chat.svelte';

	let text = $state('');
	let listEl = $state<HTMLDivElement | null>(null);

	function submit(event: SubmitEvent) {
		event.preventDefault();
		const content = text;
		text = '';
		void sendMessage(content);
	}

	// Scroll to the newest reply as the agent answers.
	$effect(() => {
		const count = chatState.items.length;
		const node = listEl;
		if (!node || count === 0) return;
		node.scrollTop = node.scrollHeight;
	});
</script>

<section class="card bg-base-200" aria-labelledby="chat-title">
	<div class="card-body gap-3 p-4">
		<div class="flex items-baseline justify-between gap-2">
			<h2 id="chat-title" class="font-display text-base font-semibold">Ngobrol sama ANABUL</h2>
			<span class="text-xs opacity-70">contoh: "kopi 25rb"</span>
		</div>

		{#if chatState.items.length > 0}
			<div
				bind:this={listEl}
				class="max-h-72 space-y-2 overflow-y-auto pr-1"
				role="log"
				aria-label="Percakapan dengan agent"
			>
				{#each chatState.items as message (message.id)}
					{#if message.role === 'user'}
						<p class="ml-auto max-w-[85%] w-fit rounded-2xl rounded-br-md bg-primary px-3 py-2 text-sm font-medium text-primary-content">
							{message.text}
						</p>
					{:else}
						<div class="max-w-[92%] space-y-2">
							<p
								class="w-fit rounded-2xl rounded-bl-md px-3 py-2 text-sm whitespace-pre-line {message.failed
									? 'bg-error/15 text-error-ink'
									: 'bg-base-100'}"
							>
								{message.text}
							</p>

							{#if message.pending && !message.pending.done}
								<div class="card border border-warning/50 bg-warning/10">
									<div class="card-body gap-2 p-3">
										<p class="text-xs font-semibold tracking-wide text-warning-content uppercase">Butuh konfirmasi</p>
										<p class="text-sm">{message.pending.summary}</p>
										<div class="flex flex-wrap gap-2">
											<button
												type="button"
												class="btn btn-warning btn-sm min-h-11"
												disabled={chatState.sending}
												onclick={() => void confirmAction(message)}
											>
												Ya, lanjutkan
											</button>
											<button
												type="button"
												class="btn btn-ghost btn-sm min-h-11"
												disabled={chatState.sending}
												onclick={() => void cancelAction(message)}
											>
												Batal
											</button>
										</div>
									</div>
								</div>
							{/if}

							{#if message.undo}
								<button
									type="button"
									class="btn btn-outline btn-sm min-h-11"
									disabled={message.undo.used}
									onclick={() => void undoTransaction(message)}
								>
									{message.undo.used ? 'Sudah dibatalkan' : 'Undo catatan ini'}
								</button>
							{/if}

							{#if message.failed}
								<button type="button" class="btn btn-ghost btn-sm min-h-11" onclick={() => void retryLastMessage()}>
									Coba kirim lagi
								</button>
							{/if}
						</div>
					{/if}
				{/each}

				{#if chatState.sending}
					<p class="w-fit rounded-2xl rounded-bl-md bg-base-100 px-3 py-2 text-sm opacity-70" aria-live="polite">
						ANABUL lagi mikir...
					</p>
				{/if}
			</div>
		{/if}

		<form class="flex items-end gap-2" onsubmit={submit}>
			<label class="sr-only" for="chat-input">Tulis catatan atau pertanyaan</label>
			<textarea
				id="chat-input"
				bind:value={text}
				rows="1"
				placeholder={'ketik aja, mis. "kopi 25rb"'}
				class="textarea textarea-bordered min-h-11 w-full resize-none text-base leading-snug"
				onkeydown={(event) => {
					if (event.key === 'Enter' && !event.shiftKey) {
						event.preventDefault();
						const content = text;
						text = '';
						void sendMessage(content);
					}
				}}
			></textarea>
			<button
				type="submit"
				class="btn btn-primary min-h-11 px-4"
				disabled={chatState.sending || text.trim().length === 0}
			>
				Kirim
			</button>
		</form>

		<p class="text-xs opacity-70">
			ANABUL mencatat lewat tool yang divalidasi. Aksi berisiko minta konfirmasi dulu.
		</p>
	</div>
</section>
