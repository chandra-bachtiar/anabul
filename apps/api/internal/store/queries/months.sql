-- name: GetMonthClosing :one
SELECT month_start, closed_at, reopened_at, snapshot
FROM month_closings
WHERE month_start = $1;

-- name: UpsertMonthClosing :one
INSERT INTO month_closings (month_start, closed_at, snapshot)
VALUES ($1, now(), $2)
ON CONFLICT (month_start) DO UPDATE
SET closed_at = now(), reopened_at = NULL, snapshot = EXCLUDED.snapshot
RETURNING month_start, closed_at, reopened_at, snapshot;

-- name: ReopenMonthClosing :one
UPDATE month_closings
SET reopened_at = now()
WHERE month_start = $1
RETURNING month_start, closed_at, reopened_at, snapshot;

-- name: ListMonthClosings :many
SELECT month_start, closed_at, reopened_at
FROM month_closings
WHERE closed_at IS NOT NULL
  AND (reopened_at IS NULL OR reopened_at < closed_at)
ORDER BY month_start DESC;
