-- name: ListCategories :many
SELECT id, name, keywords, created_at
FROM categories
ORDER BY name;

-- name: GetCategory :one
SELECT id, name, keywords, created_at
FROM categories
WHERE id = $1;

-- name: FindCategoryByName :one
SELECT id, name, keywords, created_at
FROM categories
WHERE lower(name) = lower($1);

-- name: CreateCategory :one
INSERT INTO categories (name, keywords)
VALUES ($1, $2)
RETURNING id, name, keywords, created_at;

-- name: UpdateCategory :one
UPDATE categories
SET
  name = COALESCE(sqlc.narg('name')::text, name),
  keywords = COALESCE(sqlc.narg('keywords')::text[], keywords)
WHERE id = @id
RETURNING id, name, keywords, created_at;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;
