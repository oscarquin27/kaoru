-- name: UpsertUser :exec
INSERT INTO app_user (id, username, display_name, email, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (id) DO UPDATE
    SET username     = excluded.username,
        display_name = excluded.display_name,
        email        = excluded.email,
        updated_at   = excluded.updated_at
WHERE (app_user.username, app_user.display_name, app_user.email)
          IS DISTINCT FROM (excluded.username, excluded.display_name, excluded.email);

-- name: ListUsers :many
SELECT id, username, display_name
FROM app_user
WHERE sqlc.arg(query)::text = ''
   OR username ILIKE '%' || sqlc.arg(query)::text || '%'
   OR display_name ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY display_name, username
LIMIT sqlc.arg(max_rows);
