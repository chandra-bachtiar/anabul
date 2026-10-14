# Keputusan dan Pertanyaan Terbuka

## Sudah diputuskan (oleh pemilik)
| Topik | Keputusan |
|---|---|
| Input harian | sesederhana mungkin; chat/bot-style, tanpa form berat |
| Fitur inti | catat pengeluaran, hutang piutang, kebutuhan bulanan, budgeting + sisa budget, tutup bulan |
| Pendekatan | seperti "agent harness": memahami maksud dari bahasa natural |
| Penyimpanan | database (PostgreSQL), bukan Google Sheets |
| Backend | Go |
| Frontend | Svelte (SvelteKit) |
| Struktur repo | monorepo |
| Output akhir | satu Docker image |
| Klien LLM | API OpenAI-compatible |
| Cakupan UI | versi mobile saja (web mobile-first/PWA) |
| Cara mulai | simple, bertahap (bukan Next.js) |
| Nama app | ANABUL (Asisten Ngatur Anggaran Bulanan) |
| Gaya UI | playful, maskot kucing (menggantikan usulan minimalist dark) |
| Ikon | bergambar kucing, dibuat dengan bantuan AI; versi awal SVG/PNG ada di `apps/web/static/` dan boleh disempurnakan nanti |

## Asumsi (usulan, belum dikonfirmasi pemilik)
| Topik | Asumsi | Alasan |
|---|---|---|
| Bot | Telegram | tanpa proses approval bisnis seperti WhatsApp |
| Palet dan font | krem hangat + oranye kucing; Nunito untuk isi, Fredoka/Baloo 2 untuk judul | cocok dengan maskot dan tetap terbaca |
| Library UI | Tailwind + daisyUI | cepat, ada tema dan komponen progress |
| Frontend disajikan | di-embed ke binary Go | satu image, satu port |
| Aturan sisa budget | hangus saat tutup bulan (tanpa rollover) | paling sederhana |
| Tutup bulan | bisa dibuka kembali dengan konfirmasi | aman jika ada koreksi |
| Auth | satu password + cookie sesi | pengguna tunggal |

## Pertanyaan terbuka
- Model LLM dan provider spesifik untuk `LLM_BASE_URL`/`LLM_MODEL`
- Perlu fitur akun/dompet (saldo per akun) di versi awal atau ditunda?
- Hutang piutang: otomatis membuat transaksi saat dibayar atau terpisah dari arus kas?
- Tempat hosting image (VPS, platform container) dan cara HTTPS untuk webhook

## Log keputusan baru
Tambahkan baris baru di sini setiap ada perubahan: tanggal, keputusan, alasan, dampak ke dokumen lain.

- 2026-10-13: nama app ditetapkan ANABUL dan gaya UI diubah dari minimalist dark ke playful dengan maskot kucing. Dampak: `UI_SPEC.md` ditulis ulang, `AGENT_TOOLS.md` (persona), `AGENTS.md` (aturan brand), `PRD.md`, `ARCHITECTURE.md` (nama image), aset ikon ditambahkan.
- 2026-10-13: scaffold Fase 0 selesai (struktur monorepo, web SPA, Go server, migrasi, Docker). Keputusan teknis: modul Go `anabul` dengan toolchain Go 1.26 (`golang:1.26-alpine`) karena `goose` v3.29 mensyaratkan Go 1.26; migrasi `goose` di-embed ke binary sehingga image runtime hanya berisi binary; `sqlc` di-generate lewat image Docker `sqlc/sqlc` (build dari source di Windows lambat karena cgo). Dampak: `ARCHITECTURE.md` (poin image), `README.md`.
- 2026-10-13: implementasi Fase 1-6 selesai: tools layer (validasi + aturan bulan terkunci), auth cookie sesi, REST API lengkap, agent LLM dengan registry tool dan alur konfirmasi `pending_actions`, parser sederhana sebagai cadangan saat `LLM_API_KEY` kosong, scheduler kebutuhan bulanan, webhook Telegram, dan UI mobile. Keputusan teknis: seluruh logika bisnis di `internal/tools` dan dipakai bersama REST API dan agent; agent dapat dipakai tanpa LLM supaya aplikasi tetap berguna saat key belum diisi; test regresi pesan -> tool memakai `internal/agent/testdata/messages.json`. Dampak: `ROADMAP.md`, `API.md` (tambahan endpoint `/api/auth/me`, `/api/summary`, `/api/needs/generate`, `/api/needs/:id/unpay`, `/api/months`).
- 2026-10-13: database untuk pemakaian harian diputuskan **Neon** (hosted), bukan container. Alasan: pemilik tidak ingin database berjalan di container; image jadi benar-benar stateless dan cuma perlu `DATABASE_URL`. Konsekuensi teknis: `sslmode=require` wajib, endpoint `-pooler` (PgBouncer) dideteksi otomatis di `cmd/server` dan memakai `QueryExecModeExec` supaya tidak kena masalah prepared statement, dan `docker-compose.yml` turun peran jadi alat dev saja. Dampak: `ARCHITECTURE.md` (bagian Database), `.env.example`, `README.md`.
- 2026-10-13: token warna teks disesuaikan demi WCAG AA. Oranye `#FF9F43` hanya 2,6:1 di atas krem dan mint `#3CC48B` hanya 2,2:1 di atas putih, jadi teks memakai varian turunan `--color-primary-ink`, `--color-success-ink`, `--color-error-ink`, sementara warna terang aslinya tetap dipakai untuk isian progress. Garis input juga dinaikkan kontrasnya lewat `--color-outline-warm`. Alasan: audit axe-core menemukan pelanggaran kontras pada angka pemasukan/pengeluaran. Dampak: `apps/web/src/app.css`, `UI_SPEC.md`.
- 2026-10-13: alias import frontend mengikuti SvelteKit 3, yaitu `#lib` dari `package.json` (`imports`), bukan `$lib` yang sudah dihapus. TypeScript tetap perlu `paths` di `tsconfig.json` karena peta `imports` tidak melakukan resolusi ekstensi. Dampak: `apps/web/tsconfig.json`, seluruh komponen.
- 2026-10-13: service worker disesuaikan setelah insiden halaman kosong. Penyebabnya dua hal sekaligus: server menjawab `index.html` untuk aset yang sudah tidak ada (browser menolak HTML sebagai modul JS), dan service worker menyajikan shell lama dari cache. Perbaikan: permintaan aset yang hilang dijawab 404, `index.html` dan `sw.js` dikirim dengan `Cache-Control: no-cache`, cache SW naik ke `anabul-v2`, navigasi memakai jaringan dulu, dan hanya `/_app/immutable/` (nama ber-hash) yang di-cache. Dampak: `apps/api/internal/webui/embed.go`, `apps/web/static/sw.js`.
- 2026-10-13: password dipindah dari environment ke tabel `app_settings` supaya bisa diganti dari dalam aplikasi. `APP_PASSWORD_HASH` tetap jadi sumber awal (di-seed saat server pertama menyala), dan dipakai lagi kalau baris di database dihapus. Password default `anabul123` dianggap "masih default": selama itu terdeteksi, `login`/`me` mengembalikan `using_default_password` sehingga Home menampilkan saran ganti. Aturan password baru: minimal 8 karakter, tidak sama dengan yang sekarang, tidak sama dengan default. Dampak: migrasi `00003_settings.sql`, package `internal/auth`, `API.md`, `.env.example`, `README.md`, `UI_SPEC.md`.
- 2026-10-13: backup dan restore memakai `pg_dump`/`psql` dari container `postgres:16-alpine` (`internal/pgdump` + `cmd/backup` + `cmd/restore`) supaya jalan sama di Windows, macOS, dan Linux tanpa memasang klien Postgres. Host `localhost` pada `DATABASE_URL` otomatis diganti `host.docker.internal` karena kliennya berjalan di container, dan berkas yang gagal dibuat dihapus supaya tidak ada backup palsu. Retensi `BACKUP_KEEP` (default 14) dan folder `backups/` masuk `.gitignore`. Dampak: `Makefile` (`make backup`, `make restore`), `README.md`, `ARCHITECTURE.md`.
- 2026-10-13: deployment HTTPS memakai Caddy di `docker-compose.prod.yml` (sertifikat Let's Encrypt otomatis, HSTS di proxy), dengan `deploy/Caddyfile.local` (tls internal, port 8443) untuk uji lokal. Container app diberi `TRUST_PROXY=true`, dan aplikasi hanya mempercayai `X-Forwarded-For`/`X-Forwarded-Proto` saat flag itu aktif supaya rate limit tidak bisa dilewati dengan header palsu. Webhook Telegram didaftarkan lewat `cmd/telegram-webhook` (`make webhook`, `make telegram-info`) yang mengirim `secret_token` dan membatasi `allowed_updates` ke pesan. Dampak: `docker-compose.prod.yml`, `deploy/`, `internal/telegram`, `internal/http/middleware.go`, `Makefile`.
- 2026-10-13: hardening lanjutan. CSP ketat `script-src 'self'` plus hash SHA-256 skrip inline SvelteKit yang dihitung dari `index.html` saat start (`internal/webui/csp.go`), jadi tidak perlu `unsafe-inline` untuk skrip; header lain (`nosniff`, `no-referrer`, `frame-ancestors 'none'`, `Permissions-Policy`) ikut dipasang. Rotasi `SESSION_SECRET` didukung lewat `SESSION_SECRET_PREVIOUS` (cookie lama tetap valid sampai kedaluwarsa). `ALERT_WEBHOOK_URL` mengirim notifikasi 5xx dan kegagalan start, dengan penahanan 5 menit per pesan. Dampak: `internal/http/middleware.go`, `internal/alert`, `internal/config`, `.env.example`, `README.md`.
- 2026-10-14: form tambah dan ubah transaksi di web mendapat kolom tanggal (`occurred_at` sudah didukung API sejak awal), dengan tombol cepat "Hari ini" dan "Kemarin". Saat mengubah, jam aslinya dipertahankan dan hanya tanggalnya yang berubah; saat menambah dan tanggalnya berada di bulan lain, pengguna diberi tahu lewat toast supaya tidak bingung mencari catatannya. Saat sedang membuka bulan lain, tanggal awal form diarahkan ke tanggal 1 bulan itu. Dampak: `apps/web/src/routes/transaksi/+page.svelte`, `apps/web/src/lib/format.ts`, `UI_SPEC.md`, `PRD.md`.
- 2026-10-15: pertanyaan terbuka soal arus kas hutang piutang diputuskan: **pembayaran otomatis membuat transaksi** (pengeluaran untuk hutang, pemasukan untuk piutang, `source='debt'`) supaya total bulan ikut benar, dengan saklar "Catat juga di buku harian" di form pembayaran dan `record_transaction: false` di API/tool untuk mematikannya. Dampak: `internal/tools/debts.go`, `internal/http/handlers_plan.go`, `internal/agent/tools.go`, `API.md`, `AGENT_TOOLS.md`, `PRD.md`, `UI_SPEC.md`.
- 2026-10-15: keterangan waktu di pesan didukung di parser sederhana ("kemarin", "tadi"/"hari ini", "tanggal N"/"tgl N") dan di tool agent lewat argumen `date`. Aturan "tanggal N" memakai bulan sebelumnya kalau harinya sudah lewat, karena catatan biasanya ditulis setelah kejadian; kata waktu juga dibuang dari catatan. Dampak: `internal/agent/fallback.go`, `internal/agent/tools.go`, `internal/agent/prompt.go`, `testdata/messages.json`, `AGENT_TOOLS.md`.
- 2026-10-15: halaman Hutang diperjelas: kartu menampilkan jumlah pembayaran dan tanggal bayar terakhir (query `ListDebts`/`GetDebt` menghitung `payment_count` dan `last_payment_at`), riwayat pembayaran selalu tampil di rincian (termasuk saat masih kosong), dan sheet pembayaran menutup setelah tersimpan supaya tidak terasa menggantung. Dampak: `internal/store/queries/debts.sql`, `internal/tools/debts.go`, `apps/web/src/routes/hutang/+page.svelte`, `UI_SPEC.md`.
- 2026-10-15: gerak di UI dinaikkan dari MOTION 1 ke **MOTION 2**: transisi halaman, daftar muncul berurutan, angka sisa budget "pop" saat berubah, umpan balik tekan, konfeti singkat untuk momen senang, dan haptic ringan saat aksi berhasil. Semua mematuhi `prefers-reduced-motion`, jadi pengguna yang memilih kurangi gerak tidak melihat animasi apa pun. Sekaligus stub `doc.go` sisa scaffold dihapus (dokumentasi paket sudah ada di file masing-masing) dan target `make sqlc` memakai image Docker supaya tidak perlu memasang sqlc lokal. Dampak: `apps/web/src/app.css`, `apps/web/src/lib/haptic.ts`, `apps/web/src/lib/components/Confetti.svelte`, `apps/web/src/routes/+layout.svelte`, `Makefile`, `UI_SPEC.md`.
