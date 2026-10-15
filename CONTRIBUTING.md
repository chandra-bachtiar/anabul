# Panduan Kontribusi

Senang kalau kamu mau ikut merapikan ANABUL — issue dan pull request dalam bahasa Indonesia
atau Inggris dua-duanya diterima.

Sebelum menulis kode, baca **`AGENTS.md`**. Berkas itu berisi aturan wajib repo ini (untuk
manusia maupun AI agent), dan `docs/` adalah sumber kebenaran untuk produk, skema, dan API.

## Yang kamu butuh

- Docker (untuk Postgres dev, build image, dan `make sqlc`)
- Go 1.26+ dan Node.js 22+ kalau mau menjalankan tanpa Docker
- Satu Postgres untuk test kalau menyentuh layer database

## Menjalankan di lokal

```sh
make dev-db      # Postgres lokal di container (port 5432, user/password: app)
make dev-api     # http://localhost:8080
make dev-web     # http://localhost:5173, /api sudah diproksi ke 8080
```

`make dev-web` paling enak untuk pekerjaan tampilan karena ada hot reload. Kalau mau memakai
database Neon saat ngoding, isi `DATABASE_URL` di `.env` dan lewati `make dev-db`.

Belum ada `.env`? `cp .env.example .env` lalu isi `SESSION_SECRET` dengan kalimat acak. Untuk
lokal, password defaultnya `anabul123`.

## Test

```sh
make test        # go test ./... + svelte-check
```

Test yang butuh database hanya jalan kalau `ANABUL_TEST_DATABASE_URL` diisi, contoh:

```sh
ANABUL_TEST_DATABASE_URL=postgres://app:app@localhost:5432/finance?sslmode=disable make test
```

Test itu membersihkan datanya sendiri, jadi aman dijalankan berulang di database khusus test.
Jangan arahkan ke database yang isinya kamu sayangi.

## Aturan yang paling sering dilanggar

- **Logika bisnis ada di `internal/tools`, bukan di prompt atau di UI.** Model bahasa hanya
  memilih tool dan mengisi argumen; validasi dan perhitungan ada di Go.
- **Uang selalu integer rupiah.** Jangan pakai float untuk nominal.
- **Waktu memakai zona `Asia/Jakarta`** dan disimpan sebagai `TIMESTAMPTZ`; batas bulan
  dihitung dengan zona itu.
- **Bulan yang sudah ditutup bersifat read-only**, hanya boleh dibuka lewat `reopen_month`
  dengan konfirmasi.
- **Aksi berisiko wajib lewat alur konfirmasi** (`close_month`, `reopen_month`, hapus data).
- **Jangan pernah commit secret.** API key LLM dan token Telegram tidak boleh sampai ke
  frontend, dan `.env` tidak boleh ikut ter-commit.
- **Skema hanya berubah lewat migration baru.** Migration lama jangan diedit.
- **Kode berbahasa Inggris, teks aplikasi berbahasa Indonesia.** Nama variabel, komentar,
  pesan log, dan keluaran CLI memakai bahasa Inggris; microcopy UI, balasan agent, dan pesan
  error yang tampil ke pengguna tetap bahasa Indonesia.
- **Komentar seperlunya.** Hanya untuk alasan, batasan, kasus tepi, atau perilaku yang tidak
  terlihat dari kode, maksimal dua baris.

## Query SQL dan migration

Query ditulis di `apps/api/internal/store/queries/*.sql`, kode Go-nya digenerate `sqlc`:

```sh
make sqlc        # menjalankan sqlc lewat image Docker
```

Jangan mengedit berkas `internal/store/*.sql.go` dengan tangan; ubah query-nya lalu generate
ulang.

Migration baru ditambahkan di `apps/api/migrations/` dengan nomor urut berikutnya
(`00004_...sql`). Aplikasi menjalankan migration otomatis saat start, dan manual lewat
`make migrate`.

## Alur pull request

1. Fork repo (atau buat branch kalau kamu punya akses), beri nama yang jelas seperti
   `fix/budget-copy` atau `feat/export-csv`.
2. Commit kecil dan fokus; pesan commit imperatif dalam bahasa Inggris
   (`fix(api): reject negative budget`).
3. Jalankan `make test` dan pastikan hijau.
4. Buka pull request dengan deskripsi: apa yang berubah, kenapa, dan cara mengujinya.
   Kalau menyentuh tampilan, sertakan tangkapan layar sebelum dan sesudah.
5. Kalau keputusan desain berubah, perbarui dokumen di `docs/` pada pull request yang sama
   (terutama `docs/DECISIONS.md`).

CI menjalankan `gofmt`, `go vet`, `go test`, `svelte-check`, build frontend, dan build image
Docker; pull request di-merge setelah itu hijau.
