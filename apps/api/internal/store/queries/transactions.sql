-- name: ListTransactions :many
SELECT
  t.id,
  t.type,
  t.amount,
  t.note,
  t.category_id,
  c.name AS category_name,
  t.account_id,
  a.name AS account_name,
  t.occurred_at::timestamptz AS occurred_at,
  t.month_start,
  t.source
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN accounts a ON a.id = t.account_id
WHERE t.deleted_at IS NULL
  AND t.month_start = $1
  AND (sqlc.narg('category_id')::bigint IS NULL OR t.category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('q')::text IS NULL OR t.note ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('cursor')::bigint IS NULL OR t.id < sqlc.narg('cursor'))
ORDER BY t.occurred_at DESC, t.id DESC
LIMIT $2;

-- name: GetTransaction :one
SELECT
  t.id,
  t.type,
  t.amount,
  t.note,
  t.category_id,
  c.name AS category_name,
  t.account_id,
  a.name AS account_name,
  t.occurred_at::timestamptz AS occurred_at,
  t.month_start,
  t.source
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN accounts a ON a.id = t.account_id
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- name: LastTransaction :one
SELECT
  t.id,
  t.type,
  t.amount,
  t.note,
  t.category_id,
  c.name AS category_name,
  t.account_id,
  a.name AS account_name,
  t.occurred_at::timestamptz AS occurred_at,
  t.month_start,
  t.source
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN accounts a ON a.id = t.account_id
WHERE t.deleted_at IS NULL
ORDER BY t.id DESC
LIMIT 1;

-- name: CreateTransaction :one
INSERT INTO transactions (type, amount, note, category_id, account_id, occurred_at, month_start, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
SET
  type = COALESCE(sqlc.narg('type')::text, type),
  amount = COALESCE(sqlc.narg('amount')::bigint, amount),
  note = COALESCE(sqlc.narg('note')::text, note),
  category_id = CASE
    WHEN sqlc.arg('clear_category')::bool THEN NULL
    ELSE COALESCE(sqlc.narg('category_id')::bigint, category_id)
  END,
  account_id = CASE
    WHEN sqlc.arg('clear_account')::bool THEN NULL
    ELSE COALESCE(sqlc.narg('account_id')::bigint, account_id)
  END,
  occurred_at = COALESCE(sqlc.narg('occurred_at')::timestamptz, occurred_at),
  month_start = COALESCE(sqlc.narg('month_start')::date, month_start)
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTransaction :exec
UPDATE transactions
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: MonthTotals :one
SELECT
  COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS income,
  COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS expense,
  COUNT(*)::bigint AS tx_count
FROM transactions
WHERE deleted_at IS NULL AND month_start = $1;

-- name: MonthExpenseByCategory :many
SELECT
  t.category_id,
  COALESCE(c.name, '')::text AS category_name,
  COALESCE(SUM(t.amount), 0)::bigint AS spent
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.deleted_at IS NULL AND t.month_start = $1 AND t.type = 'expense'
GROUP BY t.category_id, c.name
ORDER BY spent DESC;

-- name: DetachCategoryFromTransactions :exec
UPDATE transactions SET category_id = NULL WHERE category_id = $1;
