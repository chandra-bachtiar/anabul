# API REST

Semua di bawah `/api`, JSON, butuh cookie sesi kecuali disebut lain. Bulan memakai format `YYYY-MM`. Nominal integer rupiah. Error: `{ "error": "pesan" }` dengan status HTTP yang sesuai.

## Auth
- `POST /api/auth/login` `{ password }` -> set cookie sesi
- `POST /api/auth/logout`
- `GET /healthz` (tanpa auth)

## Dashboard
- `GET /api/dashboard?month=YYYY-MM` -> total income/expense, sisa budget per kategori, 5 transaksi terakhir, kebutuhan belum dibayar, ringkasan hutang piutang

## Transaksi
- `GET /api/transactions?month=&category_id=&q=&cursor=`
- `POST /api/transactions` `{ type, amount, note, category_id?, account_id?, occurred_at? }`
- `PATCH /api/transactions/:id`
- `DELETE /api/transactions/:id` (soft delete)
- Semua tulis ditolak (`409`) jika bulan terkunci

## Kategori dan akun
- `GET/POST /api/categories`, `PATCH/DELETE /api/categories/:id`
- `GET/POST /api/accounts`

## Budget
- `GET /api/budgets?month=` -> budget + terpakai + sisa + status
- `PUT /api/budgets` `{ month, items: [{ category_id, amount }] }`
- `POST /api/budgets/copy` `{ from, to }`

## Kebutuhan bulanan
- `GET/POST /api/recurring-needs`, `PATCH/DELETE /api/recurring-needs/:id`
- `GET /api/needs?month=`
- `POST /api/needs/:id/pay` -> tandai paid + buat transaksi

## Hutang piutang
- `GET /api/debts?status=` -> beserta sisa
- `POST /api/debts`
- `POST /api/debts/:id/payments` `{ amount, paid_at? }`
- `GET /api/debts/:id`

## Tutup bulan
- `GET /api/months/:month/preview` -> ringkasan sebelum ditutup
- `POST /api/months/:month/close` (butuh `confirm: true`)
- `POST /api/months/:month/reopen` (butuh `confirm: true`)

## Agent
- `POST /api/chat` `{ message }` -> `{ reply, pending_action?: { id, summary } }`
- `POST /api/actions/:id/confirm` dan `POST /api/actions/:id/cancel`

## Telegram
- `POST /telegram/webhook` (tanpa cookie; verifikasi header secret token dan chat id pemilik)
