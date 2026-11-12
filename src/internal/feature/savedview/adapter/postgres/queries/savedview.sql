-- name: InsertView :exec
INSERT INTO saved_view (id, owner_id, screen, name, query, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListViews :many
SELECT id, name, query, created_at
FROM saved_view
WHERE owner_id = $1
  AND screen = $2
ORDER BY lower(name), name;

-- name: CountViews :one
SELECT count(*)
FROM saved_view
WHERE owner_id = $1
  AND screen = $2;

-- name: DeleteView :execrows
DELETE
FROM saved_view
WHERE id = $1
  AND owner_id = $2;
