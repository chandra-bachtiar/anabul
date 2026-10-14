# Agent dan Tools

## Prinsip
- LLM memilih tool dan mengisi argumen; **semua logika dan validasi ada di Go**.
- Klien LLM: endpoint OpenAI-compatible (`/chat/completions`) dengan `tools` (`type: function`). `arguments` dari model adalah string JSON: decode defensif, jika rusak kembalikan error terstruktur ke model.
- Loop maksimal 5 putaran; jika lewat, balas pesan gagal yang ramah.
- Model kecil sering salah format nominal ("25.000"): normalisasi di tool (`strip non-digit`) dan tegaskan di deskripsi "integer rupiah tanpa titik/koma".

## Konteks tiap request (system prompt)
- Tanggal hari ini dan zona waktu Asia/Jakarta
- Daftar kategori, akun, nama orang di hutang piutang yang masih terbuka
- Kebutuhan bulanan yang belum dibayar
- Status bulan berjalan (terbuka/ditutup)
- Beberapa pesan terakhir dari `chat_messages` (untuk koreksi "yang tadi...")
- Persona: kamu ANABUL, asisten kucing yang ramah. Balas singkat, bahasa Indonesia santai, boleh sesekali "meong" dan maksimal satu emoji. Nominal dan status selalu ditulis jelas di awal. Untuk error, konfirmasi aksi berisiko, dan tutup bulan, hilangkan candaan dan utamakan kejelasan.
- Tanya balik satu pertanyaan pendek jika info kurang, jangan menebak

## Kebijakan aksi
| Jenis | Contoh | Perilaku |
|---|---|---|
| Baca | `get_budget_remaining`, `get_summary`, `list_debts` | langsung jalan |
| Tulis biasa | `add_transaction`, `add_debt`, `add_debt_payment`, `mark_need_paid`, `set_budget` | langsung jalan, balas konfirmasi singkat + opsi undo |
| Berisiko | `close_month`, `reopen_month`, `delete_transaction`, ubah data bulan tertutup | tidak langsung dieksekusi: tool menyimpan `pending_actions` dan mengembalikan `needs_confirmation`; eksekusi hanya setelah user konfirmasi (tombol/balasan "ya") |

## Daftar tool
| Tool | Parameter | Catatan |
|---|---|---|
| `add_transaction` | `amount:int`, `note:string`, `type: expense\|income`, `category?:string`, `account?:string`, `date?:string` | kategori kosong -> coba cocokkan keyword, selain itu NULL; `date` diisi `YYYY-MM-DD` kalau pengguna menyebut waktu lain |
| `update_transaction` | `id` atau `last:true`, field yang diubah | untuk "yang tadi salah" |
| `delete_transaction` | `id` atau `last:true` | berisiko |
| `get_summary` | `month?`, `group_by? category` | pemasukan, pengeluaran, per kategori |
| `set_budget` | `category`, `amount`, `month?` | |
| `get_budget_remaining` | `category?`, `month?` | tanpa kategori = semua |
| `add_debt` | `person`, `direction: hutang\|piutang`, `amount`, `note?`, `due_date?` | |
| `add_debt_payment` | `person` atau `debt_id`, `amount`, `date?`, `record_transaction?` | jika ambigu (banyak hutang) tanya balik; pembayaran otomatis dibuatkan transaksi kecuali `record_transaction: false` |
| `list_debts` | `status?`, `person?` | sisa, jumlah pembayaran, dan tanggal bayar terakhir ikut dikembalikan |
| `list_monthly_needs` | `month?`, `status?` | |
| `mark_need_paid` | `name` atau `id`, `amount?` | membuat transaksi pengeluaran |
| `preview_close_month` | `month?` | baca saja: ringkasan sebelum tutup |
| `close_month` | `month` | berisiko, butuh konfirmasi |
| `reopen_month` | `month` | berisiko, butuh konfirmasi |

### Keterangan waktu di pesan
Parser sederhana (dipakai kalau `LLM_API_KEY` belum diisi) mengenali "kemarin", "tadi"/"hari ini", dan "tanggal N"/"tgl N" untuk mengisi `occurred_at`. Aturan "tanggal N": kalau harinya belum lewat di bulan berjalan, dipakai bulan ini; kalau sudah lewat, dipakai bulan sebelumnya karena catatan biasanya ditulis setelah kejadian. Kata waktu dibuang dari catatan, jadi "kopi kemarin 25rb" tersimpan sebagai pengeluaran "kopi" bertanggal kemarin. Model LLM mengisi kolom yang sama lewat argumen `date`.

## Bentuk hasil tool
Selalu JSON kecil dan jelas, mis. `{"ok":true,"id":42,"amount":25000}` atau `{"error":"kategori tidak ditemukan","available":["makan-minum","transport"]}` atau `{"needs_confirmation":true,"action_id":"...","summary":"Tutup bulan 2026-10?"}`.

## Logging
Setiap pesan, tool call (nama, args, hasil), dan balasan ditulis ke `agent_logs`. Dipakai untuk debug, evaluasi, dan undo.

## Evaluasi
Simpan set contoh pesan -> tool call yang diharapkan di `apps/api/internal/agent/testdata/` (lihat tabel contoh di `PRD.md`) dan jalankan sebagai test regresi dengan LLM yang di-mock, ditambah smoke test manual dengan model asli.
