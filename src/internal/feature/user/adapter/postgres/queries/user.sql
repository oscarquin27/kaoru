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
