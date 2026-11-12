-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: InsertComponent :exec
INSERT INTO component (id, code, system_id, kind, name, description, retired_at, created_by, created_at, updated_at)
VALUES ($1, $2, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)),
        $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateComponent :execrows
UPDATE component
SET kind        = $2,
    name        = $3,
    description = $4,
    retired_at  = $5,
    updated_at  = $6
WHERE id = $1;

-- name: GetComponentByCode :one
SELECT c.id, c.code, c.kind, c.name, c.description, c.retired_at,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM component c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN app_user u ON u.id = c.created_by
WHERE c.code = $1;

-- name: ListComponentsByCodes :many
SELECT c.id, c.code, c.kind, c.name, c.description, c.retired_at,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM component c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN app_user u ON u.id = c.created_by
WHERE c.code = ANY (sqlc.arg(codes)::text[])
ORDER BY c.code;

-- name: ListComponents :many
SELECT c.id, c.code, c.kind, c.name, c.description, c.retired_at,
       c.created_by, u.display_name AS created_by_name, c.created_at, c.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM component c
         JOIN system_under_test s ON s.id = c.system_id
         JOIN app_user u ON u.id = c.created_by
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(kind)::text IS NULL OR c.kind = sqlc.narg(kind))
  AND (sqlc.narg(query)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(query) || '%' OR c.code = sqlc.narg(query))
  AND (sqlc.arg(include_retired)::boolean OR c.retired_at IS NULL)
ORDER BY c.name, c.code
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountComponents :one
SELECT count(*)
FROM component c
         JOIN system_under_test s ON s.id = c.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(kind)::text IS NULL OR c.kind = sqlc.narg(kind))
  AND (sqlc.narg(query)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(query) || '%' OR c.code = sqlc.narg(query))
  AND (sqlc.arg(include_retired)::boolean OR c.retired_at IS NULL);

-- name: LockComponent :one
-- Bloquea la fila hasta el fin de la transacción: dos escrituras sobre la misma entidad van una tras otra.
SELECT id
FROM component
WHERE code = $1
    FOR UPDATE;
