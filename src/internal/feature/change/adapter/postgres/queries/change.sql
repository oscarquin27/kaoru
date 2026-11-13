-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: InsertChange :exec
INSERT INTO change (id, code, title, description, system_id, route, state_id, window_start, window_end,
                    created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)),
        $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateChange :execrows
UPDATE change
SET title        = $2,
    description  = $3,
    route        = $4,
    state_id     = $5,
    window_start = $6,
    window_end   = $7,
    updated_at   = $8
WHERE id = $1;

-- name: GetChangeByCode :one
SELECT c.id, c.code, c.title, c.description, c.route, c.window_start, c.window_end,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN workflow_state st ON st.id = c.state_id
         JOIN app_user u ON u.id = c.created_by
WHERE c.code = $1;

-- name: ListChanges :many
SELECT c.id, c.code, c.title, c.description, c.route, c.window_start, c.window_end,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN workflow_state st ON st.id = c.state_id
         JOIN app_user u ON u.id = c.created_by
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(query)::text IS NULL OR c.title ILIKE '%' || sqlc.narg(query) || '%' OR c.code = sqlc.narg(query))
ORDER BY c.created_at DESC, c.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountChanges :one
SELECT count(*)
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN workflow_state st ON st.id = c.state_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(query)::text IS NULL OR c.title ILIKE '%' || sqlc.narg(query) || '%' OR c.code = sqlc.narg(query));

-- name: LockChange :one
-- Bloquea la fila hasta el fin de la transacción: dos escrituras sobre la misma entidad van una tras otra.
SELECT id
FROM change
WHERE code = $1
    FOR UPDATE;

-- name: ListWindows :many
-- Las ventanas de cambio vigentes: las de los CHG con ventana completa que no están en un estado final.
SELECT c.id, c.code, c.title, c.description, c.route, c.window_start, c.window_end,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN workflow_state st ON st.id = c.state_id
         JOIN app_user u ON u.id = c.created_by
WHERE c.window_start IS NOT NULL
  AND c.window_end IS NOT NULL
  AND NOT st.is_final
ORDER BY c.window_start, c.code;
