-- name: InsertComment :exec
INSERT INTO comment (id, entity_type, entity_id, body, author_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListComments :many
SELECT c.id, c.body, c.author_id, u.display_name AS author_name, c.created_at
FROM comment c
         JOIN app_user u ON u.id = c.author_id
WHERE c.entity_type = $1
  AND c.entity_id = $2
ORDER BY c.created_at, c.id;

-- name: GetUserName :one
SELECT display_name
FROM app_user
WHERE id = $1;
