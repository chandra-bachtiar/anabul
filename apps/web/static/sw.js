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

	// Data keuangan selalu diambil dari server supaya tidak pernah basi.
	if (url.pathname.startsWith('/api/') || url.pathname === '/healthz') return;

	// Aset build Vite punya nama ber-hash, jadi isinya tidak pernah berubah.
	if (url.pathname.startsWith('/_app/immutable/')) {
		event.respondWith(
			caches.match(event.request).then((tersimpan) => {
				if (tersimpan) return tersimpan;
				return fetch(event.request).then((respons) => {
					if (respons.ok) {
						const salinan = respons.clone();
						caches.open(CACHE).then((cache) => cache.put(event.request, salinan));
					}
					return respons;
				});
			})
		);
		return;
	}

	// Navigasi: utamakan jaringan supaya build baru langsung terpakai, dan
	// simpan salinannya hanya sebagai cadangan saat offline.
	if (event.request.mode === 'navigate') {
		event.respondWith(
			fetch(event.request)
				.then((respons) => {
					if (respons.ok) {
						const salinan = respons.clone();
						caches.open(CACHE).then((cache) => cache.put('/', salinan));
					}
					return respons;
				})
				.catch(() => caches.match('/'))
		);
		return;
	}

	// Berkas lain: jaringan dulu, cache hanya sebagai cadangan.
	event.respondWith(fetch(event.request).catch(() => caches.match(event.request)));
});
