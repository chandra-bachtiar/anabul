# UI Spec ANABUL (mobile-first, playful)

**ANABUL** = *Asisten Ngatur Anggaran Bulanan*. Maskotnya kucing oranye. Kesan: ramah, lucu, tapi angka tetap jelas dan cepat dibaca.

## Prinsip
- Hanya layout mobile, satu kolom. Di desktop tampilan ditaruh di tengah (`max-w-md`).
- **Playful tapi tidak mengorbankan keterbacaan angka.** Kelucuan ada di maskot, warna, bentuk membulat, dan microcopy, bukan di tabel atau nominal.
- Nominal selalu besar, tebal, kontras tinggi, `font-variant-numeric: tabular-nums`.
- Warna bermakna: hijau = aman, kuning = hampir habis, merah muda-merah = lewat/telat. Jangan hanya warna: tambah ikon/teks.
- Area tap minimal 44px. Bottom tab bar untuk navigasi utama. Tabel jadi daftar kartu.
- Hormati `prefers-reduced-motion` (matikan konfeti dan animasi maskot).

## Identitas
- Nama: ANABUL. Nama tampil: "ANABUL". Tagline: "Asisten Ngatur Anggaran Bulanan".
- Maskot: kucing oranye bulat dengan koin emas (ikon aplikasi).
- Suara: santai, hangat, sesekali "meong", maksimal satu emoji per pesan. Jangan memakai gaya lucu untuk pesan error/uang yang kritis; di sana utamakan kejelasan.

## Aset (sudah ada di `apps/web/static/`)
| File | Pakai untuk |
|---|---|
| `icon.svg`, `icon-192.png`, `icon-512.png` | ikon PWA (`purpose: any`) |
| `icon-maskable.svg`, `icon-maskable-512.png` | ikon PWA (`purpose: maskable`, ada safe zone) |
| `apple-touch-icon.png` | ikon iOS home screen |
| `favicon.svg` | favicon |
| `mascot-happy.svg` | maskot mood normal |
| `mascot-worried.svg` | maskot mood budget menipis/lewat |
| `mascot-asleep.svg` | maskot untuk empty state / bulan terkunci |

Ini versi awal buatan AI (SVG vektor, mudah diubah). Boleh diganti ilustrasi final nanti selama nama file dan fungsinya sama.

## Palet (token)
| Token | Light | Dark |
|---|---|---|
| `--bg` | `#FFF8EC` krem hangat | `#1E1712` cokelat gelap |
| `--surface` | `#FFFFFF` | `#2A211A` |
| `--ink` | `#3B2A20` | `#FFF1DE` |
| `--primary` (oranye kucing) | `#FF9F43` | `#FF9F43` |
| `--primary-strong` | `#E8812A` | `#FFB45E` |
| `--pink` (aksen) | `#FF8FA3` | `#FF8FA3` |
| `--ok` mint | `#3CC48B` | `#3CC48B` |
| `--warn` kuning | `#FFC145` | `#FFC145` |
| `--danger` | `#F0566A` | `#FF7285` |
| `--info` | `#5BB5F0` | `#5BB5F0` |

Catatan: teks di atas tombol oranye memakai `--ink` (gelap), bukan putih, supaya kontras cukup. Cek kontras sebenarnya saat implementasi (target WCAG AA). Mode dark didukung dan mengikuti sistem, dapat diganti di Pengaturan.

### Token turunan untuk teks (hasil audit kontras)

Warna di tabel atas dipilih untuk permukaan dan isian, bukan untuk teks di atas krem/putih.
Saat implementasi ditambahkan turunannya supaya semua teks lolos WCAG AA (4,5:1):

| Token | Light | Dark | Alasan |
|---|---|---|---|
| `--primary-ink` | `#B8531A` | `#FFC178` | oranye primary hanya 2,6:1 di atas krem |
| `--success-ink` | `#197A53` | `#5ED6A4` | mint terang hanya 2,2:1 di atas putih |
| `--error-ink` | `#C02A3E` | `#FF8F9F` | merah terang hanya 3,4:1 di atas putih |
| `--outline-warm` | `#8B6F5E` | `#8B6F5E` | batas input butuh 3:1 (WCAG 1.4.11) |

Warna terang aslinya tetap dipakai untuk isian progress dan latar, karena di sana
yang dinilai adalah bentuk, bukan teks, dan angkanya selalu didampingi teks.

### Arah desain saat mengerjakan UI

Reading as: aplikasi keuangan pribadi untuk satu pemilik, dipakai di HP, dengan bahasa
visual hangat dan playful ala maskot kucing, dial **ENERGY 2 / RHYTHM 2 / MOTION 2**.

- ENERGY 2: maskot, oranye hangat, dan bentuk membulat memberi karakter, tapi angka tetap fokus utama.
- RHYTHM 2: kartu sisa budget sengaja paling menonjol di Home; bagian lain menyusul dengan bobot lebih kecil.
- MOTION 2: transisi halaman yang halus, daftar yang muncul berurutan, umpan balik tekan, dan satu momen perayaan singkat saat aksi berhasil. Semua berhenti saat `prefers-reduced-motion` aktif.

Motif identitas: atap rumah dengan dua telinga kucing pada ikon navigasi Home, dan mood
maskot yang mengikuti kondisi budget. Ikon navigasi digambar sendiri dalam satu bahasa
bentuk (garis 1,8px, ujung membulat) supaya tidak terbaca sebagai set ikon bawaan dan
serasi dengan bentuk maskot.

### Gerak dan umpan balik
- Transisi halaman: fade singkat 180ms saat rute berganti (`+layout.svelte`).
- Daftar utama muncul berurutan (kelas `anim-daftar`, jeda 30ms per baris).
- Angka sisa budget "pop" saat nilainya berubah (kelas `anim-pop`).
- Tombol mengecil sedikit saat ditekan, kartu hutang terangkat 2px saat disorot.
- Momen senang (kebutuhan lunas, hutang lunas, bulan ditutup): konfeti singkat 1,4 detik dari komponen `Confetti.svelte`, dimatikan otomatis kalau pengguna memilih kurangi gerak.
- Haptic ringan (`navigator.vibrate`) saat catatan/pembayaran tersimpan, dan pola sedikit lebih panjang saat hutang lunas atau bulan ditutup. Tidak jalan kalau perangkat tidak mendukung atau kurangi gerak aktif.

## Tipografi dan bentuk
- Font membulat yang ramah: **Nunito** (isi) dan **Fredoka** atau **Baloo 2** (judul/angka besar). Self-host `woff2` (tanpa CDN) supaya PWA bisa offline dan sesuai CSP. Verifikasi dukungan angka tabular; kalau kurang rapi, pakai angka dari font fallback.
- Radius besar (`rounded-2xl`/`3xl`), bayangan lembut, tombol "pill". Boleh border tipis hangat; hindari border hitam tebal ala neobrutalism.
- Tema daisyUI kustom `anabul` (light) dan `anabul-dark` yang memetakan token di atas.

## Maskot dan interaksi
- **Home**: kucing di header dengan mood mengikuti status bulan: senang (terpakai < 80% dari total budget), khawatir (>= 80% atau ada kategori lewat), tidur (bulan terkunci atau empty state).
- Microcopy status budget (angka tetap tampil duluan):
  - aman: "Aman, meong~"
  - hampir habis: "Tinggal 20%, hati-hati ya"
  - lewat: "Waduh, kelewatan Rp 45.000"
- Empty state: maskot tidur + ajakan singkat ("Belum ada catatan. Ketik 'kopi 25rb' buat mulai.").
- Momen senang (tandai lunas, hutang lunas, tutup bulan sukses): animasi kecil/konfeti singkat, bisa dimatikan.
- Haptic ringan saat sukses mencatat (jika didukung perangkat).

## Navigasi (bottom tab)
Home | Transaksi | Budget | Hutang | Lainnya
Bilah input chat tersedia di Home dan dapat dibuka dari tab lain lewat tombol mengambang.

## Halaman
1. **Home**
   - Header ANABUL + maskot mood
   - Kartu besar: sisa budget total bulan ini + progress
   - Progress per kategori (3-5 teratas yang paling terpakai)
   - Kebutuhan bulanan belum dibayar/segera jatuh tempo (tap untuk tandai lunas)
   - 5 transaksi terakhir
   - Input chat: placeholder "ketik aja, mis. kopi 25rb"; balasan tampil sebagai gelembung ringkas dengan tombol Undo / Konfirmasi
2. **Transaksi**: daftar per hari (kartu), filter bulan/kategori, pencarian, tap untuk edit/hapus, tombol tambah manual. Form tambah dan ubah sama-sama punya kolom **tanggal catatan** (bisa mundur, ada tombol cepat "Hari ini" dan "Kemarin") supaya catatan yang lupa dimasukkan kemarin tetap bisa dicatat di hari yang benar. Kalau tanggalnya pindah bulan dari yang sedang dibuka, muncul toast yang menyebut tujuan bulannya.
3. **Budget**: pilih bulan, kategori dengan budget/terpakai/sisa/progress, edit nominal, "salin dari bulan lalu".
4. **Hutang**: dua segmen (Hutang saya / Piutang saya); kartu per orang dengan sisa, jatuh tempo, jumlah pembayaran dan tanggal bayar terakhir, riwayat bayar; tombol catat pembayaran. Form pembayaran punya saklar "Catat juga di buku harian" (bawaan aktif) supaya uang yang keluar/masuk ikut terhitung di bulan itu, dan sheet-nya menutup setelah tersimpan.
5. **Lainnya**: Kebutuhan bulanan, Tutup bulan, Kategori & akun, Pengaturan (tema, ganti password, logout).
6. **Tutup bulan** (alur khusus): pratinjau ringkasan -> cek akhir (kebutuhan belum dibayar, hutang jatuh tempo) -> konfirmasi -> hasil snapshot. Bulan tertutup memunculkan lencana "Terkunci" (maskot tidur) dan tombol "Buka kembali" dengan konfirmasi.
7. **Login**: satu input password, maskot menyapa.
8. **Pengaturan - Ganti password**: password sekarang, password baru, dan ulangan; aturan minimal 8 karakter ditulis apa adanya. Selama password masih `anabul123`, Home menampilkan kartu saran ganti (maskot khawatir) yang mengarah ke bagian ini, dan login pertama juga menyisipkan satu toast pengingat. Setelah diganti, kartu itu hilang sendiri.

## Empty/error state
- Daftar kosong: maskot tidur + ajakan aksi singkat.
- Error jaringan: pesan jelas (tanpa bercanda) + tombol coba lagi; teks chat tidak hilang saat gagal kirim.
- Bulan terkunci: semua aksi tulis dinonaktifkan dengan penjelasan.

## PWA
`manifest.webmanifest`:
```json
{
  "name": "ANABUL",
  "short_name": "ANABUL",
  "description": "Asisten Ngatur Anggaran Bulanan",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#FFF8EC",
  "theme_color": "#FF9F43",
  "icons": [
    { "src": "/icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any" },
    { "src": "/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any" },
    { "src": "/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```
Tambahkan `<link rel="icon" href="/favicon.svg">` dan `<link rel="apple-touch-icon" href="/apple-touch-icon.png">`.

## Format
Rupiah `Rp 25.000` (locale id-ID), tanggal `13 Okt 2026`.
