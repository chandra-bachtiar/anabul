-- name: ListCategories :many
SELECT id, name, keywords, created_at
FROM categories
ORDER BY name;

-- name: CreateCategory :one
INSERT INTO categories (name, keywords)
VALUES ($1, $2)
RETURNING id, name, keywords, created_at;
