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
SESSION_SECRET_PREVIOUS=
TRUST_PROXY=false
ALERT_WEBHOOK_URL=
PUBLIC_URL=
DOMAIN=
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=
LLM_MODEL=gpt-4o-mini
TELEGRAM_TOKEN=
TELEGRAM_WEBHOOK_SECRET=
TELEGRAM_OWNER_CHAT_ID=
```

Makna tambahan: `SESSION_SECRET_PREVIOUS` menerima cookie yang ditandatangani secret lama
saat rotasi, `TRUST_PROXY` mengizinkan header `X-Forwarded-*` dipakai (cookie `Secure` dan
rate limit per IP asli), `ALERT_WEBHOOK_URL` menerima notifikasi 5xx dan kegagalan start,
`PUBLIC_URL`/`DOMAIN` dipakai `make webhook` dan compose produksi.

## Database
Aplikasi tidak menyimpan state di container, semuanya di PostgreSQL yang ditunjuk
`DATABASE_URL`. Tiga pilihan yang didukung:

| Pilihan | Kapan dipakai | Catatan |
|---|---|---|
| Neon (hosted) | pemakaian sehari-hari | wajib `sslmode=require`; host `-pooler` otomatis memakai mode query tanpa prepared statement |
| Postgres terpasang di mesin | paling cepat, tanpa container | backup manual |
| `docker compose up -d postgres` | ngoprek dan uji migrasi dari DB kosong | data di volume `pgdata` |

Pindah penyedia cukup mengganti `DATABASE_URL`; tidak ada kode yang perlu diubah.
Migrasi goose dijalankan otomatis saat server start (dan lewat `make migrate`), jadi
database baru tidak butuh langkah manual. Saat database belum siap (container baru
naik atau Neon baru bangun dari idle), server menunggu sampai 10 percobaan ping
sebelum menyerah.

## Build dan deploy
Dockerfile multi-stage:
1. `node` : `npm ci && npm run build` (output statis)
2. `golang` : salin hasil build ke `apps/api/internal/webui/dist`, `CGO_ENABLED=0 go build`
3. `distroless/static:nonroot` : hanya binary (migrasi `goose` di-embed ke binary)

Nama image: `anabul`. Image stateless; state hanya di PostgreSQL (lokal via compose atau hosted seperti Neon, cukup ganti `DATABASE_URL`). Karena database tidak lagi ikut
container, `docker run` cukup diberi `--env-file .env`. Webhook Telegram butuh HTTPS publik (reverse proxy seperti Caddy/Traefik atau tunnel). Pastikan versi base image dicek ulang saat implementasi.

## Rilis
Tag `v*` yang dipush memicu workflow `Release`: image dibangun dari `Dockerfile` yang sama
dengan CI, dipush ke GHCR sebagai `ghcr.io/chandra-bachtiar/anabul:<tag>`, lalu sebuah
GitHub Release dibuat otomatis dengan catatan yang digenerate. Tag bersufiks seperti
`v0.1.0-alpha` ditandai sebagai prerelease. Versi mengikuti semver 0.x selama aplikasi
masih dipakai satu orang, dan riwayat perubahan tetap ditulis manual di `CHANGELOG.md`.

## Observabilitas dan backup
- Log terstruktur (`slog`), tabel `agent_logs` untuk jejak agent
- Header pengaman di semua respons: CSP (skrip inline SvelteKit diizinkan lewat hash yang
  dihitung dari `index.html` saat start), `X-Content-Type-Options`, `Referrer-Policy`,
  `frame-ancestors 'none'`, `Permissions-Policy`
- `ALERT_WEBHOOK_URL` mengirim pesan singkat saat ada respons 5xx atau server gagal start,
  dengan penahanan 5 menit untuk pesan yang sama
- Backup: `make backup` menjalankan `pg_dump` dari container `postgres:16-alpine` dan
  menyimpan hasil gzip di `backups/` (retensi `BACKUP_KEEP`, default 14). `make restore`
  mengembalikannya lewat `psql`. Keduanya membaca `DATABASE_URL` dari environment atau
  `.env`, dan otomatis mengganti host `localhost` menjadi `host.docker.internal` karena
  kliennya berjalan di container
- Backup terjadwal: cron (Linux/macOS) atau Task Scheduler (Windows), contohnya di README.
  Di Neon, point-in-time recovery tetap disediakan penyedia

## Deploy dengan HTTPS
`docker-compose.prod.yml` menjalankan dua service: `app` (image `anabul`, tanpa port publik)
dan `caddy` (port 80/443, sertifikat Let's Encrypt otomatis, HSTS). Container app diberi
`TRUST_PROXY=true` agar cookie sesi ditandai `Secure` dan rate limit memakai IP asli.

Untuk uji lokal dipakai `deploy/Caddyfile.local` dengan `tls internal` di port 8443
(sertifikat self-signed). Webhook Telegram didaftarkan lewat `make webhook`, yang memanggil
`setWebhook` dengan `secret_token` dan membatasi `allowed_updates` ke pesan saja; status
webhook bisa dilihat dengan `make telegram-info`.
