# ANABUL — Asisten Ngatur Anggaran Bulanan

Aplikasi keuangan pribadi (single user) yang mobile-first. Input utama berupa chat
bahasa natural ("kopi 25rb", "gua minjem 200rb ke Budi") yang diproses agent LLM
menjadi pemanggilan tool backend.

Fitur inti: catat pengeluaran tanpa ribet, hutang piutang, kebutuhan bulanan,
budgeting + sisa budget, dan tutup bulan.

## Stack

- **Backend**: Go (`apps/api`), PostgreSQL + `pgx` + `sqlc` + `goose`
- **Frontend**: SvelteKit + TypeScript, `adapter-static` (SPA) + Tailwind + daisyUI
- **Agent**: API OpenAI-compatible (`LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`)
- **Input kedua**: bot Telegram (webhook)
- **Deploy**: satu Docker image (frontend di-embed ke binary Go)

Dokumen spesifikasi ada di `docs/` — itu sumber kebenaran. Mulai dari
`docs/PRD.md`, `docs/ARCHITECTURE.md`, dan `docs/ROADMAP.md`.

## Struktur

```
apps/web/   SvelteKit (build statis -> di-embed)
apps/api/   Go server (REST, agent, webhook, scheduler)
  cmd/      server, migrate
  internal/ config, http, webui, store (sqlc), agent, tools, sched
  migrations/ goose (di-embed ke binary)
docs/       spesifikasi
```

## Prasyarat

- Go 1.26+ (go.mod memakai toolchain ini)
- Node.js 22+ (dites di Node 24)
- Docker + Docker Compose (untuk Postgres dev dan image akhir)
- `make` (opsional; di Windows jalankan lewat Git Bash, atau pakai perintah
  langsung di bawah)
- `sqlc` untuk regenerate store (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`).
  Alternatif tanpa install, pakai image resmi:
  `docker run --rm -v "<path-absolut>/apps/api:/src" -w /src sqlc/sqlc generate`

## Menjalankan di lokal

```sh
cp .env.example .env        # opsional, ada default yang cocok dengan compose
make dev-db                 # postgres via docker compose
make dev-api                # http://localhost:8080 (migrasi jalan otomatis)
make dev-web                # http://localhost:5173 (proxy /api + /healthz ke 8080)
```

Tanpa `make`:

```sh
docker compose up -d postgres
cd apps/api && go run ./cmd/server
cd apps/web && npm install --include=dev && npm run dev
```

Catatan Windows: kalau environment punya `NODE_ENV=production` global, npm akan
melewati devDependencies — pakai `npm install --include=dev`.

## Migrasi

Migrasi di-embed ke binary dan dijalankan otomatis saat server start. Untuk
menjalankannya manual:

```sh
make migrate
```

Skema DB hanya berubah lewat file baru di `apps/api/migrations/`, jangan ubah
migrasi yang sudah dipakai.

## Build dan Docker

```sh
make test        # go test + svelte-check
make build       # build web -> embed -> bin/anabul
make docker      # docker build -t anabul .
docker run --rm -p 8080:8080 -e DATABASE_URL=... anabul
```

## Konfigurasi

Semua lewat environment, lihat `.env.example`. Secret (API key LLM, token
Telegram) tidak pernah dikirim ke frontend dan tidak boleh di-commit.
