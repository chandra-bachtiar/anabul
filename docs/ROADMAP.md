# Roadmap (urutan kerja)

Prinsip: mulai kecil, sistem sudah terpakai dari awal, agent LLM dipasang terakhir di atas tool yang sudah jalan. Centang saat selesai.

## Fase 0: Skeleton end-to-end
- [x] Struktur monorepo, `Makefile`, `.env.example`
- [x] SvelteKit `adapter-static` + halaman "hello"
- [x] Go server: `/healthz`, koneksi `pgx`, migration runner `goose`, embed frontend
- [x] Dockerfile multi-stage berjalan; image menyajikan halaman + `/healthz`
- [x] `docker-compose.yml` untuk Postgres dev

## Fase 1: Catat transaksi
- [x] Migration: `accounts`, `categories`, `transactions` + seed
- [x] Auth password + sesi (cookie HttpOnly, HMAC, rate limit 10 percobaan / 15 menit)
- [x] Endpoint transaksi + `sqlc` (daftar, tambah, ubah, hapus soft delete, filter bulan/kategori/cari)
- [x] Parser sederhana untuk Telegram dan cadangan web (`kopi 25000`)
- [x] UI: Home (ringkas + obrolan) + Transaksi (daftar per hari, filter, tambah manual)

## Fase 2: Budget
- [x] Migration `budgets`
- [x] Endpoint budget + perhitungan sisa (`GET/PUT /api/budgets`, `POST /api/budgets/copy`)
- [x] UI Budget + progress di Home (status aman / hampir habis / lewat)

## Fase 3: Kebutuhan bulanan
- [x] Migration `recurring_needs`, `monthly_needs`
- [x] Scheduler generate awal bulan (+ generate manual untuk bulan berjalan)
- [x] Endpoint + UI; "bayar" membuat transaksi dan menandai lunas

## Fase 4: Hutang piutang
- [x] Migration `debts`, `debt_payments`
- [x] Endpoint + UI (dua segmen, riwayat pembayaran, status lunas otomatis)

## Fase 5: Tutup bulan
- [x] Migration `month_closings`
- [x] Preview, close, reopen + penguncian tulis di layer tool/API
- [x] Generate bulan berikutnya (budget disalin + kebutuhan dari template)
- [x] UI alur tutup bulan (pratinjau, konfirmasi, hasil snapshot, tombol buka kembali)

## Fase 6: Agent
- [x] Klien LLM OpenAI-compatible + loop tool calling (maksimal 5 putaran)
- [x] Registry tool membungkus logika fase 1-5
- [x] Konteks sistem, riwayat pendek, `agent_logs`, `chat_messages`
- [x] Alur konfirmasi (`pending_actions`) dengan tombol Ya/Batal di UI
- [x] Parser sederhana Telegram digantikan agent saat LLM tersedia, kotak chat web terhubung
- [x] Test regresi pesan -> tool call (`internal/agent/testdata/messages.json`) + test table-driven tools (nominal, bulan, budget, tutup bulan)

## Fase 7: Polesan dan operasional
- [x] PWA (manifest, ikon, service worker; data `/api` tidak pernah di-cache)
- [x] Rate limit login dan cookie sesi (HttpOnly, SameSite Lax, Secure mengikuti HTTPS)
- [x] Ganti password dari dalam aplikasi (Lainnya -> Pengaturan) + saran ganti selama masih memakai password default
- [x] Backup dan restore terjadwal (`make backup`/`make restore`, retensi otomatis, contoh cron dan Task Scheduler)
- [x] Reverse proxy + HTTPS (compose produksi + Caddy) dan pendaftaran webhook Telegram (`make webhook`)
- [x] Hardening lanjutan: CSP + security headers, rotasi `SESSION_SECRET` tanpa memaksa login ulang, notifikasi error lewat `ALERT_WEBHOOK_URL`, dan `TRUST_PROXY` agar header proxy tidak bisa dipalsukan

## Catatan status
- Semua fase (0-7) selesai. Fase 1-6 diuji dengan `go test ./...`, `svelte-check`, dan audit
  aksesibilitas (axe-core) nol pelanggaran di sembilan halaman pada tema terang dan gelap.
- Fase 7 diuji terpisah: backup dan restore dibandingkan ke database kosong (jumlah tabel dan
  baris identik), HTTPS diuji lewat Caddy dengan sertifikat internal (cookie sesi keluar
  dengan atribut `Secure`), perintah webhook Telegram diuji terhadap API tiruan, dan CSP
  dicocokkan dengan hash skrip inline yang benar-benar dilayani.
- Yang belum diuji dengan layanan sungguhan: koneksi ke Neon, dan pengiriman pesan Telegram
  asli (keduanya butuh kredensial milik pemilik).
