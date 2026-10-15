# Changelog

Semua perubahan yang layak dicatat ada di sini.
Format mengikuti [Keep a Changelog](https://keepachangelog.com/id-ID/1.1.0/) dan versi mengikuti
[Semantic Versioning](https://semver.org/lang/id/).

## [Unreleased]

## [0.1.0] - 2026-10-15

Rilis pertama.

### Ditambahkan

- Pencatatan pengeluaran dan pemasukan lewat obrolan: agent LLM dengan 14 tool, dan parser
  sederhana sebagai cadangan saat `LLM_API_KEY` belum diisi
- Keterangan waktu di pesan ("kemarin", "tadi", "tanggal 5") ikut disimpan sebagai tanggal catatan
- Hutang piutang: catat, cicil, riwayat pembayaran, dan transaksi otomatis saat pembayaran
  (pengeluaran untuk hutang, pemasukan untuk piutang)
- Kebutuhan bulanan berulang dengan jatuh tempo dan generate per bulan
- Budget per kategori dengan sisa budget, status warna, dan salin dari bulan sebelumnya
- Tutup bulan: pratinjau, snapshot, kunci bulan, buka kembali dengan konfirmasi
- Halaman mobile-first: Home, Transaksi, Budget, Hutang, Kebutuhan, Kategori dan akun,
  Pengaturan, Tutup bulan
- Bot Telegram (webhook) yang hanya melayani pemilik, diverifikasi lewat secret token
- PWA: manifest, service worker, mode gelap, animasi level 2 dengan `prefers-reduced-motion`
- Auth satu password: cookie sesi HMAC, ganti password dari dalam aplikasi, peringatan saat
  password masih default
- Backup dan restore lewat `pg_dump` di container, plus retensi berkas backup
- Deploy HTTPS satu perintah dengan Caddy (`docker-compose.prod.yml`)
- Hardening: CSP dengan hash skrip inline, rotasi `SESSION_SECRET`, rate limit login,
  notifikasi error 5xx ke webhook
