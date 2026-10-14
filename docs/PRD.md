# PRD: ANABUL (Asisten Ngatur Anggaran Bulanan)

## Tujuan
Mencatat keuangan harian dengan friksi sekecil mungkin. Pemilik app (single user) malas membuka app lalu mengisi banyak kolom, jadi input utama berupa kalimat bebas lewat chat (bot Telegram dan kotak chat di web).

## Brand
Nama: **ANABUL** (Asisten Ngatur Anggaran Bulanan). Maskot kucing, gaya playful dan ramah. Detail visual di `UI_SPEC.md`.

## Non-goals (versi awal)
- Multi-user / berbagi akun
- Aplikasi native (cukup web mobile-first dan PWA)
- Layout desktop khusus
- Integrasi otomatis ke bank/e-wallet
- Multi-currency

## Pengguna
Satu orang (pemilik). Pemakaian harian dari HP.

## Fitur dan acceptance criteria

### 1. Catat pengeluaran/pemasukan tanpa ribet
- Pesan seperti `kopi 25rb`, `makan siang 35000`, `gajian 5jt` tercatat dengan nominal, catatan, tipe, dan kategori (kategori boleh kosong).
- Balasan konfirmasi singkat dengan opsi undo.
- Jika info kurang (mis. nominal tidak ada), agent bertanya balik satu pertanyaan pendek, tidak menebak.
- Kategori dapat diisi/diubah kemudian dari UI.
- Input juga tersedia lewat form kecil di web, termasuk pilihan tanggal untuk catatan yang baru sempat dimasukkan belakangan (mis. lupa mencatat kemarin).

### 2. Hutang piutang
- Catat hutang (user berhutang) dan piutang (orang berhutang ke user): nama orang, nominal, catatan, jatuh tempo opsional.
- Catat pembayaran/cicilan; sisa dihitung dari total minus pembayaran.
- Status otomatis: berjalan / lunas.
- Daftar hutang piutang per orang beserta sisa dan jatuh tempo.
- Opsional: pembayaran dapat dikaitkan ke transaksi dan akun. (Dikerjakan: pembayaran otomatis membuat transaksi pengeluaran/pemasukan lewat `source='debt'`, bisa dimatikan dari form pembayaran atau lewat `record_transaction: false`.)

### 3. Kebutuhan bulanan
- Template kebutuhan bulanan berulang (listrik, internet, cicilan, dll) dengan nominal perkiraan dan tanggal jatuh tempo.
- Tiap awal bulan, daftar kebutuhan bulan itu digenerate otomatis dari template.
- Tandai "sudah dibayar" membuat transaksi pengeluaran terkait secara otomatis.
- Dashboard menampilkan kebutuhan yang belum dibayar dan yang segera jatuh tempo.

### 4. Budgeting dan sisa budget
- Budget per kategori per bulan; dapat disalin dari bulan sebelumnya.
- Sisa budget = budget minus total pengeluaran kategori di bulan itu (dihitung, tidak disimpan).
- Tampilan progress dengan status warna: aman / hampir habis / lewat.
- Bisa ditanyakan via chat ("sisa budget makan berapa?").

### 5. Tutup bulan
- Ringkasan bulan: total pemasukan, pengeluaran, selisih, sisa/over tiap budget, kebutuhan belum dibayar, posisi hutang piutang.
- Saat ditutup: simpan snapshot, kunci transaksi bulan itu, siapkan bulan berikutnya (budget dan kebutuhan bulanan).
- Butuh konfirmasi eksplisit. Dapat dibuka kembali (reopen) dengan konfirmasi.
- Default aturan sisa budget: hangus (tidak rollover). Lihat `DECISIONS.md`.

### 6. Agent (pemahaman maksud)
- Pesan bebas dipetakan ke tool backend lewat LLM tool calling.
- Semua perubahan data melalui tool yang divalidasi; aksi berisiko minta konfirmasi.
- Semua interaksi tercatat untuk debug dan undo.

## Contoh pesan yang harus dimengerti
| Pesan | Maksud |
|---|---|
| `kopi 25rb` | catat pengeluaran 25.000 |
| `gua minjem 200rb ke Budi` | hutang 200.000 ke Budi |
| `Budi minjem 150rb` | piutang 150.000 dari Budi |
| `Budi udah bayar 100rb` | pembayaran piutang |
| `listrik udah dibayar` | tandai kebutuhan bulanan lunas |
| `sisa budget makan berapa?` | query sisa budget |
| `bulan ini boros di mana?` | ringkasan per kategori |
| `tutup bulan` | mulai alur tutup bulan (konfirmasi) |
| `yang tadi salah, harusnya 30rb` | koreksi transaksi terakhir |

## Metrik keberhasilan
- Mencatat satu transaksi butuh < 5 detik dan satu pesan.
- Dipakai harian tanpa terasa melelahkan.
