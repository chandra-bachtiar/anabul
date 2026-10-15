# Kebijakan Keamanan

ANABUL adalah aplikasi keuangan pribadi yang dipasang sendiri (self-hosted). Celah keamanan
di kode ini bisa berdampak ke data keuangan orang yang memakainya, jadi laporanmu dihargai.

## Versi yang diperbaiki

Perbaikan dikerjakan di branch `main` dan dirilis sebagai versi terbaru. Kalau kamu memakai
versi lama, cara tercepat adalah memperbarui ke `main` atau rilis terakhir.

## Cara melaporkan

**Jangan buka issue publik untuk kerentanan.** Pakai GitHub private vulnerability reporting:

1. Buka tab **Security** di repo ini, lalu **Report a vulnerability**.
2. Jelaskan dampaknya, langkah reproduksi, dan versi/commit yang kamu pakai.
3. Kalau fitur pelaporan privat itu belum aktif, hubungi maintainer lewat pesan langsung di
   GitHub: [@chandra-bachtiar](https://github.com/chandra-bachtiar).

Sertakan kalau ada: contoh request, log, dan saran perbaikan. Laporan yang belum kamu
verifikasi tetap berguna — kirim saja.

Kami berusaha menjawab secepat yang bisa dilakukan maintainer, tanpa janji tenggat. Nama
pelapor akan disebut di catatan rilis kalau kamu mengizinkan.

## Yang dianggap kerentanan

- Bypass login atau sesi (misalnya cookie bisa dipalsukan, verifikasi HMAC bisa dilewati)
- Akses data tanpa sesi lewat endpoint apa pun, termasuk webhook Telegram
- Secret (API key LLM, token Telegram, `SESSION_SECRET`) yang bocor ke frontend, ke log, atau
  ke respons API
- SQL injection, XSS, SSRF, path traversal, atau eksekusi perintah lewat input pengguna
- Rate limit login yang bisa dilewati (misalnya lewat header `X-Forwarded-For` saat
  `TRUST_PROXY` tidak aktif)
- Kebocoran data lintas bulan terkunci: mengubah data bulan yang sudah ditutup tanpa
  `reopen_month` dan konfirmasi

## Yang bukan kerentanan

- Belum ada fitur multi-pengguna; aplikasi ini memang untuk satu orang
- Siapa pun yang punya akses fisik atau menguasai server tempat aplikasi berjalan
- Kehilangan data karena tidak ada backup yang dijalankan
- Kuota atau biaya LLM/Telegram di akunmu sendiri

## Kalau kamu memasang sendiri

- **Ganti password default.** Tutorial memakai `anabul123` supaya mudah dicoba, dan password
  itu tertulis di README publik.
- Isi `SESSION_SECRET` dengan kalimat acak yang panjang dan jangan dibagikan.
- Pakai `TRUST_PROXY=true` hanya kalau aplikasi benar-benar berada di belakang reverse proxy
  milikmu; kalau tidak, header `X-Forwarded-For` jadi bisa dipalsukan.
- Terapkan HTTPS (misalnya lewat Caddy seperti di `docker-compose.prod.yml`) sebelum
  mengaksesnya dari internet, dan batasi hanya untuk dirimu sendiri.
- Jalankan backup secara berkala; lihat bagian backup di `README.md`.
