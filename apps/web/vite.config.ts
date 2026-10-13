import adapter from '@sveltejs/adapter-static';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const apiTarget = process.env.API_URL ?? 'http://localhost:8080';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// SPA: seluruh routing ditangani di klien, fallback disajikan server Go.
			adapter: adapter({
				fallback: 'index.html'
			})
		})
	],
	server: {
		proxy: {
			'/api': apiTarget,
			'/healthz': apiTarget
		}
	}
});
