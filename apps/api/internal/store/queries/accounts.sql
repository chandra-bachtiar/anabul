-- name: ListAccounts :many
SELECT id, name, created_at
FROM accounts
ORDER BY name;

-- name: GetAccount :one
SELECT id, name, created_at
FROM accounts
WHERE id = $1;

-- name: FindAccountByName :one
SELECT id, name, created_at
FROM accounts
WHERE lower(name) = lower($1);

-- name: CreateAccount :one
INSERT INTO accounts (name)
VALUES ($1)
RETURNING id, name, created_at;
