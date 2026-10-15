## Apa yang berubah

<!-- Ringkas, satu-dua kalimat. Sebutkan halaman atau paket yang tersentuh. -->

## Kenapa

<!-- Masalah yang diselesaikan, atau tautkan issue: "Menutup #12". -->

## Cara menguji

<!-- Langkah yang bisa diikuti reviewer, mis. buka Hutang → tambah pembayaran → cek riwayat. -->

## Checklist

- [ ] `make test` hijau (atau `go test ./...` dan `npm run check` kalau tanpa `make`)
- [ ] Tidak ada secret baru yang ikut ter-commit (`.env`, token, API key)
- [ ] Skema database hanya berubah lewat migration baru
- [ ] Dokumen di `docs/` diperbarui kalau ada keputusan desain yang berubah
- [ ] Tangkapan layar dilampirkan kalau menyentuh tampilan

## Catatan tambahan

<!-- Hal yang perlu diketahui reviewer: trade-off, hal yang sengaja tidak dikerjakan, dll. -->
