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
- Hosting PostgreSQL: lokal/self-hosted, Neon, atau Supabase? (sempat dibandingkan; belum diputuskan)
- Model LLM dan provider spesifik untuk `LLM_BASE_URL`/`LLM_MODEL`
- Perlu fitur akun/dompet (saldo per akun) di versi awal atau ditunda?
- Hutang piutang: otomatis membuat transaksi saat dibayar atau terpisah dari arus kas?
- Tempat hosting image (VPS, platform container) dan cara HTTPS untuk webhook

## Log keputusan baru
Tambahkan baris baru di sini setiap ada perubahan: tanggal, keputusan, alasan, dampak ke dokumen lain.

- 2026-10-13: nama app ditetapkan ANABUL dan gaya UI diubah dari minimalist dark ke playful dengan maskot kucing. Dampak: `UI_SPEC.md` ditulis ulang, `AGENT_TOOLS.md` (persona), `AGENTS.md` (aturan brand), `PRD.md`, `ARCHITECTURE.md` (nama image), aset ikon ditambahkan.
- 2026-10-13: scaffold Fase 0 selesai (struktur monorepo, web SPA, Go server, migrasi, Docker). Keputusan teknis: modul Go `anabul` dengan toolchain Go 1.26 (`golang:1.26-alpine`) karena `goose` v3.29 mensyaratkan Go 1.26; migrasi `goose` di-embed ke binary sehingga image runtime hanya berisi binary; `sqlc` di-generate lewat image Docker `sqlc/sqlc` (build dari source di Windows lambat karena cgo). Dampak: `ARCHITECTURE.md` (poin image), `README.md`.
