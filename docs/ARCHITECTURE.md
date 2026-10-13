# Arsitektur

## Gambaran
Monorepo, dua aplikasi, satu image. Frontend Svelte dibuild jadi file statis, di-embed ke binary Go, disajikan oleh server yang sama. Backend juga menyediakan REST API, webhook Telegram, agent LLM, dan scheduler.

```
HP -> (Telegram | Web PWA)
        |                |
   /telegram/webhook   /api/*  + file statis
        \              /
         Go server --> Agent loop --> LLM (OpenAI-compatible)
              |            |
              |         Tool layer (validasi + logika bisnis)
              v
          PostgreSQL
```

## Struktur monorepo
```
apps/
  web/   SvelteKit (adapter-static, fallback index.html)
  api/
    cmd/server/main.go
    internal/
      agent/    loop LLM + registry tool + konteks
      tools/    satu file per tool
      store/    sqlc (queries + kode generate)
      http/     handler REST, auth, webhook Telegram
      sched/    cron (generate kebutuhan bulanan)
      webui/    embed.go (go:embed all:dist)
    migrations/ (goose)
```

## Backend (Go)
- Router: `net/http` (routing Go 1.22+) atau `chi`
- DB: `pgx` + `sqlc`; migration `goose` dijalankan otomatis saat startup
- Scheduler: `robfig/cron`, zona `Asia/Jakarta`
- Frontend disajikan lewat `webui.Handler()` dengan fallback ke `index.html` untuk routing SPA
- Rute: `/api/*` REST, `/telegram/webhook`, `/healthz`, sisanya file statis

## Frontend (SvelteKit)
- `adapter-static` dengan `fallback: 'index.html'` (SPA)
- Mobile-first, satu kolom, bottom tab bar, PWA (manifest + service worker)
- Memanggil `/api` dengan cookie sesi; saat dev, Vite proxy `/api` ke `localhost:8080`
- Detail tampilan: lihat `UI_SPEC.md`

## Agent LLM
- Klien tipis ke endpoint OpenAI-compatible (`/chat/completions`), `base_url`/key/model dari env
- Loop tool calling, maksimal 5 putaran per pesan
- Detail: lihat `AGENT_TOOLS.md`

## Auth
- Satu password (`APP_PASSWORD`, disimpan sebagai hash) menghasilkan cookie sesi `HttpOnly`, `Secure`, `SameSite=Lax`
- Rate limit pada endpoint login
- Webhook Telegram: verifikasi secret token header dan batasi ke `TELEGRAM_OWNER_CHAT_ID`

## Konfigurasi (env)
```
PORT=8080
TZ=Asia/Jakarta
DATABASE_URL=postgres://app:app@localhost:5432/finance?sslmode=disable
APP_PASSWORD_HASH=
SESSION_SECRET=
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=
LLM_MODEL=gpt-4o-mini
TELEGRAM_TOKEN=
TELEGRAM_WEBHOOK_SECRET=
TELEGRAM_OWNER_CHAT_ID=
```

## Build dan deploy
Dockerfile multi-stage:
1. `node` : `npm ci && npm run build` (output statis)
2. `golang` : salin hasil build ke `apps/api/internal/webui/dist`, `CGO_ENABLED=0 go build`
3. `distroless/static:nonroot` : hanya binary (migrasi `goose` di-embed ke binary)

Nama image: `anabul`. Image stateless; state hanya di PostgreSQL (lokal via compose atau hosted seperti Neon, cukup ganti `DATABASE_URL`). Webhook Telegram butuh HTTPS publik (reverse proxy seperti Caddy/Traefik atau tunnel). Pastikan versi base image dicek ulang saat implementasi.

## Observabilitas dan backup
- Log terstruktur (`slog`), tabel `agent_logs` untuk jejak agent
- Backup: `pg_dump` terjadwal jika DB self-hosted; jika hosted, gunakan fitur restore penyedia
