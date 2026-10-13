-- name: ListAccounts :many
SELECT id, name, created_at
FROM accounts
ORDER BY name;

-- name: CreateAccount :one
INSERT INTO accounts (name)
VALUES ($1)
RETURNING id, name, created_at;
