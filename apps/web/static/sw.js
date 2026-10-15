const CACHE = 'anabul-v2';
const SHELL = ['/manifest.webmanifest', '/favicon.svg', '/icon-192.png', '/icon-512.png'];

self.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(CACHE)
			.then((cache) => cache.addAll(SHELL))
			.then(() => self.skipWaiting())
	);
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key))))
			.then(() => self.clients.claim())
	);
});

self.addEventListener('fetch', (event) => {
	const url = new URL(event.request.url);
	if (event.request.method !== 'GET' || url.origin !== self.location.origin) return;

	// Financial data always comes from the server, so it can never go stale.
	if (url.pathname.startsWith('/api/') || url.pathname === '/healthz') return;

	// Vite build assets carry hashed names, so their content never changes.
	if (url.pathname.startsWith('/_app/immutable/')) {
		event.respondWith(
			caches.match(event.request).then((cached) => {
				if (cached) return cached;
				return fetch(event.request).then((response) => {
					if (response.ok) {
						const copy = response.clone();
						caches.open(CACHE).then((cache) => cache.put(event.request, copy));
					}
					return response;
				});
			})
		);
		return;
	}

	// Navigation: prefer the network so a new build takes effect right away, and
	// keep the copy only as an offline fallback.
	if (event.request.mode === 'navigate') {
		event.respondWith(
			fetch(event.request)
				.then((response) => {
					if (response.ok) {
						const copy = response.clone();
						caches.open(CACHE).then((cache) => cache.put('/', copy));
					}
					return response;
				})
				.catch(() => caches.match('/'))
		);
		return;
	}

	// Everything else: network first, cache only as the fallback.
	event.respondWith(fetch(event.request).catch(() => caches.match(event.request)));
});
