<script lang="ts">
	interface Props {
		active: string;
	}

	let { active }: Props = $props();

	const menu = [
		{ href: '/', label: 'Home' },
		{ href: '/transaksi', label: 'Transaksi' },
		{ href: '/budget', label: 'Budget' },
		{ href: '/hutang', label: 'Hutang' },
		{ href: '/lainnya', label: 'Lainnya' }
	];

	function isActive(href: string): boolean {
		if (href === '/') return active === '/' || active === '';
		return active === href || active.startsWith(`${href}/`);
	}
</script>

<nav
	aria-label="Navigasi utama"
	class="fixed inset-x-0 bottom-0 z-30 border-t border-base-300 bg-base-200/95 pb-[env(safe-area-inset-bottom)] backdrop-blur-sm"
>
	<ul class="mx-auto flex w-full max-w-md items-stretch justify-between px-2">
		{#each menu as item (item.href)}
			{@const itemActive = isActive(item.href)}
			<li class="flex-1">
				<a
					href={item.href}
					aria-current={itemActive ? 'page' : undefined}
					class="flex min-h-[56px] flex-col items-center justify-center gap-1 rounded-xl px-1 pt-2 pb-1 text-xs font-semibold transition-colors hover:bg-base-300/60 focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary {itemActive
						? 'text-primary-ink'
						: 'text-base-content/70'}"
				>
					<!-- One shape language (1.8px lines, round caps) so the icons match the mascot, not a stock set. -->
					<svg viewBox="0 0 24 24" class="h-6 w-6" aria-hidden="true" fill="none">
						{#if item.href === '/'}
							<!-- House with cat ears: the ANABUL brand motif. -->
							<path
								d="M3.5 10.2 12 3.6l8.5 6.6V19a1.4 1.4 0 0 1-1.4 1.4H4.9A1.4 1.4 0 0 1 3.5 19z"
								stroke="currentColor"
								stroke-width="1.8"
								stroke-linejoin="round"
								stroke-linecap="round"
								fill={itemActive ? 'currentColor' : 'none'}
								fill-opacity={itemActive ? 0.16 : 0}
							/>
							<path d="M7.2 6.9 6 3.6l3 1.6M16.8 6.9 18 3.6l-3 1.6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
						{:else if item.href === '/transaksi'}
							<!-- Receipt: the daily log. -->
							<path
								d="M6 3.8h12v14.9l-2-1.4-2 1.4-2-1.4-2 1.4-2-1.4-2 1.4z"
								stroke="currentColor"
								stroke-width="1.8"
								stroke-linejoin="round"
								fill={itemActive ? 'currentColor' : 'none'}
								fill-opacity={itemActive ? 0.16 : 0}
							/>
							<path d="M9 8.2h6M9 11.6h4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
						{:else if item.href === '/budget'}
							<!-- Coin stack: monthly budgets. -->
							<ellipse
								cx="12"
								cy="7"
								rx="7"
								ry="3"
								stroke="currentColor"
								stroke-width="1.8"
								fill={itemActive ? 'currentColor' : 'none'}
								fill-opacity={itemActive ? 0.16 : 0}
							/>
							<path d="M5 7v5c0 1.7 3.1 3 7 3s7-1.3 7-3V7" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
							<path d="M5 12v5c0 1.7 3.1 3 7 3s7-1.3 7-3v-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
						{:else if item.href === '/hutang'}
							<!-- Two circling arrows: money changing direction. -->
							<path d="M4.5 9.6A7.6 7.6 0 0 1 17.6 7" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
							<path d="M19.5 14.4A7.6 7.6 0 0 1 6.4 17" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
							<path d="M14.6 4.4 18 6.9l-2.7 3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" fill="none" />
							<path d="M9.4 19.6 6 17.1l2.7-3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" fill="none" />
						{:else}
							<!-- Four tiles: the remaining menus. -->
							<rect x="4.2" y="4.2" width="6.4" height="6.4" rx="2" stroke="currentColor" stroke-width="1.8" fill={itemActive ? 'currentColor' : 'none'} fill-opacity={itemActive ? 0.16 : 0} />
							<rect x="13.4" y="4.2" width="6.4" height="6.4" rx="2" stroke="currentColor" stroke-width="1.8" />
							<rect x="4.2" y="13.4" width="6.4" height="6.4" rx="2" stroke="currentColor" stroke-width="1.8" />
							<rect x="13.4" y="13.4" width="6.4" height="6.4" rx="2" stroke="currentColor" stroke-width="1.8" />
						{/if}
					</svg>
					<span>{item.label}</span>
				</a>
			</li>
		{/each}
	</ul>
</nav>
