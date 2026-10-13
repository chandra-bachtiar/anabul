# Roadmap (urutan kerja)

Prinsip: mulai kecil, sistem sudah terpakai dari awal, agent LLM dipasang terakhir di atas tool yang sudah jalan. Centang saat selesai.

## Fase 0: Skeleton end-to-end
- [x] Struktur monorepo, `Makefile`, `.env.example`
- [x] SvelteKit `adapter-static` + halaman "hello"
- [x] Go server: `/healthz`, koneksi `pgx`, migration runner `goose`, embed frontend
- [x] Dockerfile multi-stage berjalan; image menyajikan halaman + `/healthz`
- [x] `docker-compose.yml` untuk Postgres dev

## Fase 1: Catat transaksi
- [ ] Migration: `accounts`, `categories`, `transactions` + seed
- [ ] Auth password + sesi
- [ ] Endpoint transaksi + `sqlc`
- [ ] Webhook Telegram dengan parser kaku (`kopi 25000`)
- [ ] UI: Home (ringkas) + Transaksi

## Fase 2: Budget
- [ ] Migration `budgets`
- [ ] Endpoint budget + perhitungan sisa
- [ ] UI Budget + progress di Home

## Fase 3: Kebutuhan bulanan
- [ ] Migration `recurring_needs`, `monthly_needs`
- [ ] Scheduler generate awal bulan (+ generate manual untuk bulan berjalan)
- [ ] Endpoint + UI; "bayar" membuat transaksi

## Fase 4: Hutang piutang
- [ ] Migration `debts`, `debt_payments`
- [ ] Endpoint + UI

## Fase 5: Tutup bulan
- [ ] Migration `month_closings`
- [ ] Preview, close, reopen + penguncian tulis di layer tool/API
- [ ] Generate bulan berikutnya (budget + kebutuhan)
- [ ] UI alur tutup bulan

## Fase 6: Agent
- [ ] Klien LLM OpenAI-compatible + loop tool calling
- [ ] Registry tool membungkus logika fase 1-5
- [ ] Konteks sistem, riwayat pendek, `agent_logs`
- [ ] Alur konfirmasi (`pending_actions`)
- [ ] Ganti parser kaku Telegram dan hubungkan kotak chat web
- [ ] Test regresi pesan -> tool call

## Fase 7: Polesan dan operasional
- [ ] PWA (manifest, ikon, service worker)
- [ ] Backup dan restore terjadwal
- [ ] Reverse proxy + HTTPS, set webhook Telegram
- [ ] Rate limit, hardening cookie
