# AGENTS.md

Instruksi untuk AI coding agent (Claude Code, Codex, Cursor, dll) yang mengerjakan repo ini.
**Baca file ini dulu, lalu baca dokumen di `docs/` yang relevan dengan task sebelum menulis kode.**

## Ringkasan proyek
**ANABUL** (Asisten Ngatur Anggaran Bulanan), aplikasi pencatat keuangan pribadi (single user), mobile-first, dengan agent LLM yang
memahami pesan natural ("kopi 25rb", "gua minjem 200rb ke Budi") lalu memanggil tool backend.
Fitur inti: catat pengeluaran tanpa ribet, hutang piutang, kebutuhan bulanan, budgeting + sisa budget, tutup bulan.

## Stack
- Backend: Go (`apps/api`), PostgreSQL, `pgx`, `sqlc`, `goose`, `robfig/cron`
- Frontend: SvelteKit + TypeScript, `adapter-static` mode SPA (`apps/web`), Tailwind (+ daisyUI), PWA
- LLM: API OpenAI-compatible (`LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL` via env)
- Input kedua: bot Telegram (webhook)
- Deploy: satu Docker image (frontend di-embed ke binary Go via `go:embed`)

## Struktur repo
```
apps/web/    SvelteKit (build statis)
apps/api/    Go server (REST, agent, tools, webhook, scheduler, embed frontend)
docs/        spesifikasi (sumber kebenaran)
Dockerfile   multi-stage: web -> api -> distroless
docker-compose.yml  dev: postgres
Makefile
```

## Perintah (target Makefile yang harus ada)
`make dev-db`, `make dev-api`, `make dev-web`, `make migrate`, `make sqlc`, `make test`, `make build`, `make docker`

## Aturan wajib
1. **Logika bisnis ada di Go (tool layer), bukan di LLM.** LLM hanya memilih tool dan mengisi argumen.
2. **Uang = integer rupiah (`BIGINT`).** Dilarang float untuk nominal.
3. **Zona waktu `Asia/Jakarta`.** Simpan `TIMESTAMPTZ`; batas bulan dihitung dengan zona ini.
4. **Validasi semua input tool** (nominal > 0, kategori/orang ada, bulan belum ditutup). Jangan percaya output LLM.
5. **Bulan yang sudah ditutup bersifat read-only** kecuali lewat `reopen_month` dengan konfirmasi.
6. **Aksi berisiko** (`close_month`, `reopen_month`, hapus, ubah data bulan tertutup) wajib lewat alur konfirmasi (lihat `docs/AGENT_TOOLS.md`).
7. **Secret (API key LLM, token Telegram) tidak boleh sampai ke frontend** dan tidak boleh di-commit.
8. Semua endpoint (kecuali health, login, webhook Telegram yang diverifikasi) butuh sesi login. Webhook Telegram hanya menerima `TELEGRAM_OWNER_CHAT_ID`.
9. Semua tool call dan hasilnya dicatat ke `agent_logs`.
10. Skema DB hanya diubah lewat migration baru, jangan edit migration lama yang sudah dipakai.
11. Jangan menambah dependency besar tanpa alasan; prinsipnya "mulai simple".
12. Kalau keputusan desain berubah, **update dokumen di `docs/`** (terutama `DECISIONS.md`) di PR yang sama.

13. **Brand**: nama app ANABUL, maskot kucing, gaya UI playful. Ikuti `docs/UI_SPEC.md` (palet, font, maskot, microcopy). Teks UI dan balasan agent berbahasa Indonesia santai; pesan error dan uang tetap jelas, tidak dibuat bercanda. Nama image Docker: `anabul`.

14. **Bahasa kode vs bahasa tampilan.** Semua yang ada di dalam kode memakai bahasa Inggris: nama variabel, fungsi, tipe, komentar, pesan log, keluaran perintah CLI, komentar di Makefile/Docker/compose/Caddyfile, dan `.env.example`. Bahasa Indonesia hanya untuk yang dilihat pengguna: microcopy UI, balasan agent, dan pesan error yang muncul di aplikasi (termasuk body `{"error": ...}` dari REST API dan error domain di `internal/tools`). Contoh pesan di `apps/api/internal/agent/testdata/` tetap Indonesia karena itu contoh input pengguna.

## Konvensi kode
- Go: `gofmt`, error di-wrap dengan konteks, `context.Context` di semua operasi I/O, tidak ada global state; test table-driven untuk tool dan perhitungan budget/tutup bulan.
- Query SQL ditulis di `apps/api/internal/store/queries/*.sql`, kode digenerate `sqlc`.
- Svelte: TypeScript, komponen kecil, state sederhana (stores/runes), tidak ada library state tambahan kecuali perlu.
- Komentar ditulis seperti senior menulis catatan: hanya untuk alasan, batasan, kasus tepi, atau perilaku yang tidak terlihat dari kode. Maksimal dua baris, sentence case, tanpa banner, emoji, atau narasi langkah. Kalau kode sudah jelas, tidak ada komentar.
- Commit kecil dan fokus, pesan commit imperatif.

## Definition of done
- Fitur sesuai acceptance criteria di `docs/PRD.md`
- Test lulus (`make test`), migration bisa jalan dari DB kosong
- `make docker` sukses dan image jalan end-to-end
- Dokumen terkait diperbarui

## Indeks dokumen
- `docs/README.md` indeks dokumen (ringkasan bahasa Inggris)
- `docs/PRD.md` kebutuhan produk dan acceptance criteria
- `docs/ARCHITECTURE.md` arsitektur, build, deploy, env
- `docs/DATA_MODEL.md` skema database
- `docs/API.md` endpoint REST
- `docs/AGENT_TOOLS.md` desain agent dan daftar tool
- `docs/UI_SPEC.md` spesifikasi UI mobile
- `docs/ROADMAP.md` urutan kerja dan checklist
- `docs/DECISIONS.md` keputusan dan pertanyaan terbuka
