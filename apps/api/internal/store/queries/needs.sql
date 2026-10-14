-- name: ListRecurringNeeds :many
SELECT
  r.id,
  r.name,
  r.amount,
  r.due_day,
  r.category_id,
  c.name AS category_name,
  r.active
FROM recurring_needs r
LEFT JOIN categories c ON c.id = r.category_id
ORDER BY r.due_day, r.name;

-- name: GetRecurringNeed :one
SELECT
  r.id,
  r.name,
  r.amount,
  r.due_day,
  r.category_id,
  c.name AS category_name,
  r.active
FROM recurring_needs r
LEFT JOIN categories c ON c.id = r.category_id
WHERE r.id = $1;

-- name: CreateRecurringNeed :one
INSERT INTO recurring_needs (name, amount, due_day, category_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateRecurringNeed :one
UPDATE recurring_needs
SET
  name = COALESCE(sqlc.narg('name')::text, name),
  amount = COALESCE(sqlc.narg('amount')::bigint, amount),
  due_day = COALESCE(sqlc.narg('due_day')::smallint, due_day),
  category_id = COALESCE(sqlc.narg('category_id')::bigint, category_id),
  active = COALESCE(sqlc.narg('active')::bool, active)
WHERE id = @id
RETURNING *;

-- name: DeleteRecurringNeed :exec
DELETE FROM recurring_needs WHERE id = $1;

-- name: DetachCategoryFromRecurringNeeds :exec
UPDATE recurring_needs SET category_id = NULL WHERE category_id = $1;

-- name: DetachRecurringNeedFromMonthlyNeeds :exec
UPDATE monthly_needs SET recurring_need_id = NULL WHERE recurring_need_id = $1;

-- name: ListMonthlyNeeds :many
SELECT
  n.id,
  n.recurring_need_id,
  n.month_start,
  n.name,
  n.amount,
  n.due_date,
  n.status,
  n.transaction_id,
  r.category_id,
  c.name AS category_name
FROM monthly_needs n
LEFT JOIN recurring_needs r ON r.id = n.recurring_need_id
LEFT JOIN categories c ON c.id = r.category_id
WHERE n.month_start = $1
  AND (sqlc.narg('status')::text IS NULL OR n.status = sqlc.narg('status'))
ORDER BY n.due_date, n.name;

-- name: GetMonthlyNeed :one
SELECT
  n.id,
  n.recurring_need_id,
  n.month_start,
  n.name,
  n.amount,
  n.due_date,
  n.status,
  n.transaction_id,
  r.category_id,
  c.name AS category_name
FROM monthly_needs n
LEFT JOIN recurring_needs r ON r.id = n.recurring_need_id
LEFT JOIN categories c ON c.id = r.category_id
WHERE n.id = $1;

-- name: FindMonthlyNeedByName :one
SELECT
  n.id,
  n.recurring_need_id,
  n.month_start,
  n.name,
  n.amount,
  n.due_date,
  n.status,
  n.transaction_id,
  r.category_id,
  c.name AS category_name
FROM monthly_needs n
LEFT JOIN recurring_needs r ON r.id = n.recurring_need_id
LEFT JOIN categories c ON c.id = r.category_id
WHERE n.month_start = $1 AND lower(n.name) = lower($2)
ORDER BY n.id
LIMIT 1;

-- name: GenerateMonthlyNeeds :exec
INSERT INTO monthly_needs (recurring_need_id, month_start, name, amount, due_date)
SELECT
  r.id,
  $1::date,
  r.name,
  r.amount,
  (
    date_trunc('month', $1::date)
    + (
        LEAST(
          r.due_day,
          EXTRACT(
            DAY FROM (date_trunc('month', $1::date) + interval '1 month - 1 day')
          )::int
        ) - 1
      ) * interval '1 day'
  )::date
FROM recurring_needs r
WHERE r.active
ON CONFLICT (recurring_need_id, month_start) DO NOTHING;

-- name: CreateMonthlyNeed :one
INSERT INTO monthly_needs (recurring_need_id, month_start, name, amount, due_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: MarkMonthlyNeedPaid :one
UPDATE monthly_needs
SET status = 'paid', transaction_id = $2
WHERE id = $1
RETURNING *;

-- name: MarkMonthlyNeedUnpaid :one
UPDATE monthly_needs
SET status = 'unpaid', transaction_id = NULL
WHERE id = $1
RETURNING *;

-- name: CountUnpaidNeeds :one
SELECT COUNT(*)::bigint AS total
FROM monthly_needs
WHERE month_start = $1 AND status = 'unpaid';
