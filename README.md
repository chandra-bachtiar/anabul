# ANABUL

Catatan keuangan yang cukup ditulis satu kali: ketik `kopi 25rb`, beres.

ANABUL (Asisten Ngatur Anggaran Bulanan) adalah aplikasi keuangan pribadi untuk satu
orang. Input utamanya bukan form panjang, tapi obrolan kecil: "makan siang 35rb",
"gua minjem 200rb ke Budi", "listrik udah dibayar". Angka dan aturannya dikerjakan
server, bukan model bahasa, jadi catatanmu tetap konsisten walau kalimatnya bebas.

Isinya: catat pengeluaran dan pemasukan, kebutuhan bulanan yang berulang, budget per
kategori beserta sisanya, hutang piutang, dan tutup bulan.

## Isi halaman ini

- [Yang kamu butuh](#yang-kamu-butuh)
- [Pasang pakai Neon (cara yang dipakai sehari-hari)](#pasang-pakai-neon-cara-yang-dipakai-sehari-hari)
- [Password](#password)
- [Jalanin di lokal](#jalanin-di-lokal)
- [Cara ngobrol sama ANABUL](#cara-ngobrol-sama-anabul)
- [Perintah yang sering dipakai](#perintah-yang-sering-dipakai)
- [Soal database](#soal-database)
- [Backup dan restore](#backup-dan-restore)
- [Deploy dengan HTTPS](#deploy-dengan-https)
- [Keamanan](#keamanan)
- [Yang jalan di balik layar](#yang-jalan-di-balik-layar)
- [Kalau ada masalah](#kalau-ada-masalah)
- [Struktur repo](#struktur-repo)

## Yang kamu butuh

- Docker (buat build image dan, kalau mau, Postgres lokal)
- Go 1.26+ dan Node.js 22+ kalau mau ngoding langsung tanpa Docker
- Satu database Postgres. Rekomendasinya [Neon](https://neon.tech) supaya tidak ada
  database yang jalan di container
- Opsional: API key LLM (OpenAI atau endpoint apa pun yang OpenAI-compatible) dan bot
  Telegram

## Pasang pakai Neon (cara yang dipakai sehari-hari)

**1. Siapkan database di Neon**

Bikin project di [neon.tech](https://neon.tech), lalu salin connection string dari
dashboard. Bentuknya kira-kira:

```
postgresql://namauser:nopassword@ep-xxxx-pooler.ap-southeast-1.aws.neon.tech/anabul?sslmode=require
```

Dua hal yang perlu diperhatikan:

- `sslmode=require` harus ikut. Neon menolak koneksi tanpa SSL.
- Host yang mengandung `-pooler` (PgBouncer) tetap aman dipakai. Server mendeteksi
  itu dan mengirim query tanpa prepared statement, jadi tidak ada error
  "prepared statement already exists" yang sering muncul di endpoint pooler.

**2. Siapkan berkas konfigurasi**

```sh
cp .env.example .env
```

Isi `.env`:

```sh
DATABASE_URL=postgresql://...neon.tech/anabul?sslmode=require
SESSION_SECRET=kalimat-acak-panjang-bebas
APP_PASSWORD_HASH=<hasil perintah di bawah>
```

Bikin hash password:

```sh
cd apps/api && go run ./cmd/password "password-rahasia-kamu"
```

Belum mau bikin hash sendiri? Pakai password default **`anabul123`** dulu, lalu ganti dari
dalam aplikasi setelah berhasil masuk (lihat [Password](#password)).

**3. Jalankan aplikasinya**

```sh
make docker          # docker build -t anabul .
docker run -d --name anabul -p 8080:8080 --env-file .env anabul
```

Buka `http://localhost:8080`, masukkan password, selesai. Migrasi database dijalankan
sendiri saat server menyala, jadi tidak ada langkah terpisah. Kalau database baru
dibuat dan masih "bangun" dari idle, server menunggu sampai 10 percobaan sebelum
menyerah.

Mau pakai LLM dan Telegram juga? Tambahkan di `.env`:

```sh
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4o-mini
TELEGRAM_TOKEN=123456:ABC...
TELEGRAM_WEBHOOK_SECRET=rahasia-webhook
TELEGRAM_OWNER_CHAT_ID=123456789
```

`LLM_API_KEY` kosong bukan berarti aplikasinya mati. Agent otomatis memakai parser
sederhana, dan kalimat seperti `kopi 25rb` atau `Budi udah bayar 100rb` tetap tercatat.
Bedanya, kalimat yang lebih rumit tidak terbaca.

## Password

Password default aplikasi ini **`anabul123`**. Itu password untuk pemakaian sendiri di
komputer lokal, dan karena tertulis di README, jangan dibiarkan kalau aplikasinya bisa
diakses orang lain.

Cara gantinya lewat aplikasi: masuk dulu, lalu buka **Lainnya → Pengaturan → Ganti password**.
Isi password sekarang, password baru (minimal 8 karakter), dan ulangi sekali lagi. Kalau kamu
baru masuk memakai password default, Home menampilkan saran ganti sampai passwordnya benar-benar
diganti.

Password disimpan sebagai hash bcrypt di tabel `app_settings`. Saat server pertama kali
menyala, hash dari `APP_PASSWORD_HASH` disalin ke tabel itu, dan setelah kamu ganti dari
aplikasi, environment tidak perlu diubah lagi.

Kalau passwordnya lupa:

```sh
# hapus hash di database, lalu restart supaya APP_PASSWORD_HASH dipakai lagi
docker exec anabul-postgres-1 psql -U app -d finance -c "DELETE FROM app_settings WHERE key = 'password_hash'"
docker restart anabul
```

Belum ada `APP_PASSWORD_HASH`? Bikin dengan `cd apps/api && go run ./cmd/password "password-baru"`,
tempel ke `.env`, lalu jalankan ulang container-nya.

## Jalanin di lokal
Buat ngoding, dua terminal:

```sh
make dev-db            # Postgres lokal di container, khusus ngoprek
make dev-api           # http://localhost:8080
make dev-web           # http://localhost:5173, sudah proxy /api ke 8080
```

`make dev-web` lebih enak dipakai saat menyentuh tampilan karena ada hot reload.
Kalau kamu mau sekalian pakai Neon waktu ngoding, tinggal isi `DATABASE_URL` di `.env`
ke connection string Neon dan lewati `make dev-db`.

Kalau environment Windows-mu punya `NODE_ENV=production` global, npm akan melewatkan
devDependencies. Pakai `npm install --include=dev`.

## Cara ngobrol sama ANABUL

Tulis apa adanya di kolom obrolan Home (atau kirim ke bot Telegram):

| Pesan | Yang terjadi |
|---|---|
| `kopi 25rb` | pengeluaran Rp 25.000, kategori ditebak dari kata "kopi" |
| `gajian 5jt` | pemasukan Rp 5.000.000 |
| `kopi kemarin 25rb` | pengeluaran Rp 25.000 bertanggal kemarin |
| `makan malam tgl 5 40rb` | pengeluaran bertanggal 5 bulan ini (jika harinya belum lewat) |
| `gua minjem 200rb ke Budi` | tercatat hutang ke Budi |
| `Budi minjem 150rb` | tercatat piutang dari Budi |
| `Budi udah bayar 100rb` | pembayaran piutang Budi, sekaligus masuk pemasukan bulan itu |
| `listrik udah dibayar` | kebutuhan bulanan lunas, pengeluarannya ikut dibuat |
| `sisa budget makan berapa?` | sisa budget kategori makan-minum |
| `bulan ini boros di mana?` | ringkasan per kategori |
| `tutup bulan` | mulai alur tutup bulan, tetap butuh konfirmasi |
| `yang tadi salah, harusnya 30rb` | koreksi transaksi terakhir |

Aksi yang berisiko (hapus transaksi, tutup bulan, buka bulan) tidak pernah langsung
jalan. ANABUL menahan aksinya dan menampilkan tombol konfirmasi dulu.

Kalau kamu mencatat pembayaran hutang atau piutang, uangnya ikut masuk catatan harian
(pengeluaran untuk hutang, pemasukan untuk piutang) supaya total bulan itu benar. Saklar
"Catat juga di buku harian" di form pembayaran bisa dimatikan kalau kamu ingin pembayaran
dicatat terpisah dari arus kas.

Semua tombol di aplikasi juga bisa dipakai tanpa obrolan: Tambah manual di halaman
Transaksi, isi budget di halaman Budget, dan seterusnya. Kalau obrolan bukan gayamu,
aplikasinya tetap lengkap.

## Perintah yang sering dipakai

```sh
make dev-db      # postgres lokal (opsional, buat ngoprek)
make dev-api     # jalanin server Go
make dev-web     # jalanin SvelteKit dengan hot reload
make migrate     # jalankan migrasi manual
make sqlc        # regenerate kode database setelah mengubah query SQL
make test        # go test + svelte-check
make build       # build frontend, embed ke binary, hasil di bin/anabul
make docker      # build image anabul
make backup      # simpan dump database ke backups/
make restore     # kembalikan dump: make restore FILE=backups/anabul-....sql.gz
make prod-up     # jalankan app + Caddy (HTTPS) untuk produksi
make webhook     # daftarkan webhook Telegram
make telegram-info  # lihat status webhook Telegram
```

Tanpa `make` (Windows tanpa Git Bash misalnya):

```sh
docker compose up -d postgres
cd apps/api && go run ./cmd/server
cd apps/web && npm run dev
```

## Soal database

Aplikasinya tidak peduli database itu di mana, yang penting Postgres dan
`DATABASE_URL` mengarah ke sana. Tiga pilihan yang masuk akal:

**Neon (rekomendasi, tanpa database di container).** Gratis untuk pemakaian pribadi,
ada branch database, dan tidak ada yang perlu dirawat. Yang perlu diingat: Neon bisa
tertidur saat tidak dipakai, jadi permintaan pertama bisa terasa lambat beberapa detik.
Server sudah menunggu koneksi sampai 10 kali percobaan supaya itu tidak jadi error.

**Postgres lokal tanpa container.** Kalau kamu sudah punya Postgres terpasang di
komputer, arahkan saja `DATABASE_URL` ke sana. Paling cepat, tapi backup jadi
tanggung jawabmu.

**Postgres di container (buat ngoprek).** `docker compose up -d postgres` menyalakan
Postgres 16 di port 5432 dengan user `app`, password `app`, database `finance`. Ini
yang dipakai `make dev-db`, dan datanya disimpan di volume `pgdata`. Cocok untuk
menguji migrasi dari database kosong tanpa menyentuh database Neon-mu.

Kalau pindah dari container ke Neon, yang berubah hanya `DATABASE_URL`. Tidak ada
kode yang perlu disesuaikan.

Catatan backup: lihat bagian [Backup dan restore](#backup-dan-restore) di bawah. `pg_dump`
terjadwal bisa dijalankan dari komputer mana pun yang punya Docker, termasuk ke Neon.

## Backup dan restore

Backup memakai `pg_dump` dari container Postgres resmi, jadi tidak perlu memasang klien
Postgres di mesinmu:

```sh
make backup                                   # hasilnya di backups/anabul-YYYYmmdd-HHMMSS.sql.gz
BACKUP_KEEP=30 make backup                    # simpan 30 berkas terbaru (default 14)
BACKUP_DIR=D:\backup-anabul make backup       # simpan ke folder lain
make restore FILE=backups/anabul-20261014-072020.sql.gz
```

Yang perlu diingat: `restore` menimpa isi tabel yang ada, jadi arahkan `DATABASE_URL` ke
database yang benar dulu (idealnya database baru), baru jalankan restore. `DATABASE_URL`
dibaca dari environment atau dari `.env` di akar repo.

Supaya jalan sendiri, jadwalkan perintahnya:

```sh
# Linux/macOS, tiap hari jam 03:15
15 3 * * * cd /srv/anabul && /usr/bin/make backup >> /var/log/anabul-backup.log 2>&1
```

```powershell
# Windows Task Scheduler: jalankan tiap hari jam 03:15
schtasks /create /tn "ANABUL backup" /sc daily /st 03:15 `
  /tr "cmd /c cd /d E:\Project\anabul && make backup"
```

Kalau database-nya di Neon, backup tambahan seperti point-in-time recovery sudah disediakan
dashboard mereka. Backup mandiri di sini berguna sebagai salinan kedua di komputer sendiri.

## Deploy dengan HTTPS

Untuk memakai dari HP di luar rumah, aplikasi perlu domain dan HTTPS. Repo ini menyertakan
`docker-compose.prod.yml` dan Caddy sebagai reverse proxy yang mengurus sertifikat
Let's Encrypt otomatis:

```sh
# 1. Arahkan domain ke IP server (A record), lalu:
DOMAIN=anabul.contoh.com docker compose -f docker-compose.prod.yml up -d --build
```

Caddy membuka port 80 dan 443, meneruskan ke aplikasi di port internal 8080, dan container
app diberi `TRUST_PROXY=true` supaya cookie sesi ditandai `Secure` dan pembatasan percobaan
login memakai IP asli pengunjung.

Buat mencoba HTTPS di lokal tanpa domain, pakai konfigurasi Caddy bertanda tangan sendiri
(`deploy/Caddyfile.local`) dan buka `https://localhost:8443` (browser akan memperingatkan
sertifikat karena self-signed, itu wajar).

Kalau webhook Telegram ingin dipakai, aplikasi harus bisa diakses lewat HTTPS publik:

```sh
# isi TELEGRAM_TOKEN, TELEGRAM_WEBHOOK_SECRET, dan PUBLIC_URL di .env dulu
make webhook          # daftarkan https://domainmu/telegram/webhook
make telegram-info    # cek status: url, jumlah update tertunda, error terakhir
```

Telegram hanya melayani `TELEGRAM_OWNER_CHAT_ID`, dan setiap permintaan webhook diverifikasi
lewat secret token.

## Keamanan

- Semua respons membawa CSP ketat (skrip hanya dari origin sendiri, skrip inline SvelteKit
  diizinkan lewat hash), `X-Content-Type-Options`, `Referrer-Policy: no-referrer`, dan
  `frame-ancestors 'none'`. HSTS ditambahkan Caddy saat HTTPS aktif.
- Cookie sesi `HttpOnly` + `Secure` + `SameSite=Lax`, ditandatangani HMAC. Mau ganti
  `SESSION_SECRET` tanpa memaksa login ulang? Isi secret lama di `SESSION_SECRET_PREVIOUS`,
  ganti `SESSION_SECRET`, jalankan ulang, lalu hapus `SESSION_SECRET_PREVIOUS` setelah
  semua sesi lama kedaluwarsa.
- Percobaan login dibatasi 10 kali per 15 menit per IP (memakai `X-Forwarded-For` hanya
  kalau `TRUST_PROXY=true`).
- Isi `ALERT_WEBHOOK_URL` kalau mau dapat notifikasi saat server error (5xx) atau gagal
  start; pesan yang sama ditahan 5 menit supaya tidak spam.

## Yang jalan di balik layar

- **Go** untuk server: REST API, webhook Telegram, agent, dan scheduler.
- **Postgres** dengan `pgx`, query ditulis di `internal/store/queries` dan digenerate
  pakai `sqlc`.
- **SvelteKit** dengan `adapter-static` untuk tampilan mobile-first. Hasil build-nya
  di-embed ke binary Go, jadi satu image berisi semuanya.
- **Agent** memilih tool dan mengisi argumennya. Semua perhitungan, validasi, dan
  aturan (misalnya bulan yang sudah ditutup tidak bisa diubah) ada di Go. Model bahasa
  tidak pernah menyentuh database langsung.
- **Scheduler** menyiapkan daftar kebutuhan bulanan tiap awal bulan, dan sekali lagi
  saat server baru menyala.

## Kalau ada masalah

**Halaman login bilang login belum diaktifkan.** `APP_PASSWORD_HASH` dan
`SESSION_SECRET` belum diisi. Bikin hash dengan `go run ./cmd/password "..."`, isi
`.env`, lalu jalankan ulang server.

**"database tidak dapat dihubungi".** Cek `DATABASE_URL`. Untuk Neon pastikan ada
`sslmode=require` dan host-nya benar. Kalau database baru bangun dari idle, tunggu
beberapa detik lalu muat ulang.

**Balasan agent terdengar kaku dan sering tidak paham.** Berarti `LLM_API_KEY` belum
diisi dan parser sederhana yang jalan. Isi key-nya, jalankan ulang, lalu coba
kalimat yang lebih rumit.

**Tampilan seperti masih versi lama.** Aplikasi memasang service worker untuk mode
offline. Hard reload (Ctrl+Shift+R) atau tutup tabnya lalu buka lagi.

**Angka di kategori salah.** Kategori ditebak dari kata kunci catatan. Ubah kata
kuncinya di Lainnya → Kategori dan akun.

## Struktur repo

```
apps/web/    SvelteKit: halaman, komponen, PWA
apps/api/    Go: server, agent, tools, store, migrasi
docs/        spesifikasi dan keputusan
Dockerfile   multi-stage: web -> api -> distroless
```

Dokumen di `docs/` adalah sumber kebenaran proyek ini:

- `docs/PRD.md` kebutuhan produk dan acceptance criteria
- `docs/ARCHITECTURE.md` arsitektur, build, deploy, dan env
- `docs/DATA_MODEL.md` skema database
- `docs/API.md` daftar endpoint REST
- `docs/AGENT_TOOLS.md` desain agent dan daftar tool
- `docs/UI_SPEC.md` spesifikasi tampilan
- `docs/ROADMAP.md` urutan kerja dan progres
- `docs/DECISIONS.md` keputusan dan catatannya

Nama ANABUL, maskot kucing, dan gaya playful-nya ikut dijelaskan di `docs/UI_SPEC.md`.
