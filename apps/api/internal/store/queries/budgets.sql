-- name: ListBudgets :many
SELECT
  b.id,
  b.month_start,
  b.category_id,
  c.name AS category_name,
  b.amount,
  COALESCE(spent.total, 0)::bigint AS spent
FROM budgets b
JOIN categories c ON c.id = b.category_id
LEFT JOIN LATERAL (
  SELECT SUM(t.amount) AS total
  FROM transactions t
  WHERE t.deleted_at IS NULL
    AND t.month_start = b.month_start
    AND t.category_id = b.category_id
    AND t.type = 'expense'
) spent ON true
WHERE b.month_start = $1
ORDER BY c.name;

-- name: GetBudget :one
SELECT
  b.id,
  b.month_start,
  b.category_id,
  c.name AS category_name,
  b.amount,
  COALESCE(spent.total, 0)::bigint AS spent
FROM budgets b
JOIN categories c ON c.id = b.category_id
LEFT JOIN LATERAL (
  SELECT SUM(t.amount) AS total
  FROM transactions t
  WHERE t.deleted_at IS NULL
    AND t.month_start = b.month_start
    AND t.category_id = b.category_id
    AND t.type = 'expense'
) spent ON true
WHERE b.month_start = $1 AND b.category_id = $2;

-- name: UpsertBudget :one
INSERT INTO budgets (month_start, category_id, amount)
VALUES ($1, $2, $3)
ON CONFLICT (month_start, category_id) DO UPDATE SET amount = EXCLUDED.amount
RETURNING *;

-- name: CopyBudgets :exec
INSERT INTO budgets (month_start, category_id, amount)
SELECT $2::date, b.category_id, b.amount
FROM budgets b
WHERE b.month_start = $1
ON CONFLICT (month_start, category_id) DO NOTHING;

-- name: SumBudgets :one
SELECT COALESCE(SUM(amount), 0)::bigint AS total
FROM budgets
WHERE month_start = $1;

-- name: DeleteBudgetsForCategory :exec
DELETE FROM budgets WHERE category_id = $1;

-- name: CategorySpent :one
SELECT COALESCE(SUM(amount), 0)::bigint AS spent
FROM transactions
WHERE deleted_at IS NULL AND month_start = $1 AND type = 'expense' AND category_id = $2;
