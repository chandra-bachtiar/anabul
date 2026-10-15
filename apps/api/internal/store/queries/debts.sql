-- name: ListDebts :many
SELECT
  d.id,
  d.person,
  d.direction,
  d.amount_total,
  d.note,
  d.due_date,
  d.status,
  d.created_at,
  COALESCE(p.paid, 0)::bigint AS paid,
  (d.amount_total - COALESCE(p.paid, 0))::bigint AS remaining,
  COALESCE(p.payment_count, 0)::bigint AS payment_count,
  p.last_paid_at::timestamptz AS last_payment_at
FROM debts d
LEFT JOIN LATERAL (
  SELECT SUM(amount) AS paid, COUNT(*) AS payment_count, MAX(paid_at) AS last_paid_at
  FROM debt_payments WHERE debt_id = d.id
) p ON true
WHERE (sqlc.narg('status')::text IS NULL OR d.status = sqlc.narg('status'))
  AND (sqlc.narg('person')::text IS NULL OR d.person ILIKE sqlc.narg('person'))
ORDER BY d.status, d.due_date NULLS LAST, d.created_at DESC;

-- name: GetDebt :one
SELECT
  d.id,
  d.person,
  d.direction,
  d.amount_total,
  d.note,
  d.due_date,
  d.status,
  d.created_at,
  COALESCE(p.paid, 0)::bigint AS paid,
  (d.amount_total - COALESCE(p.paid, 0))::bigint AS remaining,
  COALESCE(p.payment_count, 0)::bigint AS payment_count,
  p.last_paid_at::timestamptz AS last_payment_at
FROM debts d
LEFT JOIN LATERAL (
  SELECT SUM(amount) AS paid, COUNT(*) AS payment_count, MAX(paid_at) AS last_paid_at
  FROM debt_payments WHERE debt_id = d.id
) p ON true
WHERE d.id = $1;

-- name: FindOpenDebtByPerson :many
SELECT
  d.id,
  d.person,
  d.direction,
  d.amount_total,
  d.note,
  d.due_date,
  d.status,
  d.created_at,
  COALESCE(p.paid, 0)::bigint AS paid,
  (d.amount_total - COALESCE(p.paid, 0))::bigint AS remaining
FROM debts d
LEFT JOIN LATERAL (
  SELECT SUM(amount) AS paid FROM debt_payments WHERE debt_id = d.id
) p ON true
WHERE d.status = 'open' AND lower(d.person) = lower($1)
  AND (sqlc.narg('direction')::text IS NULL OR d.direction = sqlc.narg('direction'))
ORDER BY d.created_at;

-- name: CreateDebt :one
INSERT INTO debts (person, direction, amount_total, note, due_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateDebtPayment :one
INSERT INTO debt_payments (debt_id, amount, paid_at, transaction_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDebtPayments :many
SELECT * FROM debt_payments WHERE debt_id = $1 ORDER BY paid_at DESC, id DESC;

-- name: DebtPaidTotal :one
SELECT COALESCE(SUM(amount), 0)::bigint AS paid FROM debt_payments WHERE debt_id = $1;

-- name: SetDebtStatus :exec
UPDATE debts SET status = $2 WHERE id = $1;

-- name: DebtTotals :one
SELECT
  COALESCE(SUM(d.amount_total - COALESCE(p.paid, 0)) FILTER (WHERE d.direction = 'hutang'), 0)::bigint AS debt_total,
  COALESCE(SUM(d.amount_total - COALESCE(p.paid, 0)) FILTER (WHERE d.direction = 'piutang'), 0)::bigint AS receivable_total
FROM debts d
LEFT JOIN LATERAL (
  SELECT SUM(amount) AS paid FROM debt_payments WHERE debt_id = d.id
) p ON true
WHERE d.status = 'open';
