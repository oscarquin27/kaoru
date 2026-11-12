-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: InsertPackage :exec
INSERT INTO package (id, code, title, description, system_id, state_id, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)), $5, $6, $7, $8);

-- name: UpdatePackage :execrows
UPDATE package
SET title       = $2,
    description = $3,
    state_id    = $4,
    updated_at  = $5
WHERE id = $1;

-- name: LockPackage :one
-- Bloquea la fila hasta el fin de la transacción: dos escrituras sobre el mismo paquete van una tras otra.
SELECT id
FROM package
WHERE code = $1
    FOR UPDATE;

-- name: GetPackageByCode :one
SELECT p.id, p.code, p.title, p.description, p.created_by, u.display_name AS created_by_name, p.created_at, p.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name, st.is_initial AS state_is_initial
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
         JOIN workflow_state st ON st.id = p.state_id
         JOIN app_user u ON u.id = p.created_by
WHERE p.code = $1;

-- name: ListPackages :many
SELECT p.id, p.code, p.title, p.description, p.created_by, u.display_name AS created_by_name, p.created_at, p.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name, st.is_initial AS state_is_initial
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
         JOIN workflow_state st ON st.id = p.state_id
         JOIN app_user u ON u.id = p.created_by
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(query)::text IS NULL OR p.title ILIKE '%' || sqlc.narg(query) || '%' OR p.code = sqlc.narg(query))
ORDER BY p.created_at DESC, p.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountPackages :one
SELECT count(*)
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
         JOIN workflow_state st ON st.id = p.state_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(query)::text IS NULL OR p.title ILIKE '%' || sqlc.narg(query) || '%' OR p.code = sqlc.narg(query));

-- name: ListPackageBuilds :many
-- Los BUILD de varios paquetes, por código.
SELECT pb.package_id, b.id, b.code, b.version, b.checksum
FROM package_build pb
         JOIN build b ON b.id = pb.build_id
WHERE pb.package_id = ANY (sqlc.arg(package_ids)::uuid[])
ORDER BY b.code;

-- name: ListPackageRequirements :many
-- Los REQ de varios paquetes con su estado actual, por código.
SELECT pr.package_id, r.id, r.code, r.title, st.key AS state_key, st.name AS state_name
FROM package_requirement pr
         JOIN requirement r ON r.id = pr.requirement_id
         JOIN workflow_state st ON st.id = r.state_id
WHERE pr.package_id = ANY (sqlc.arg(package_ids)::uuid[])
ORDER BY r.code;

-- name: InsertPackageBuild :exec
INSERT INTO package_build (package_id, build_id, added_by, added_at)
VALUES ($1, $2, $3, $4);

-- name: DeletePackageBuild :exec
DELETE
FROM package_build
WHERE package_id = $1
  AND build_id = $2;

-- name: InsertPackageRequirement :exec
INSERT INTO package_requirement (package_id, requirement_id, added_by, added_at)
VALUES ($1, $2, $3, $4);

-- name: DeletePackageRequirement :exec
DELETE
FROM package_requirement
WHERE package_id = $1
  AND requirement_id = $2;
