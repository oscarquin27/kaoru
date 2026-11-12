-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: InsertEnvironment :exec
INSERT INTO environment (id, code, system_id, name, kind, os, notes, retired_at, created_by, created_at, updated_at)
VALUES ($1, $2, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)), $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateEnvironment :execrows
UPDATE environment
SET name       = $2,
    kind       = $3,
    os         = $4,
    notes      = $5,
    retired_at = $6,
    updated_at = $7
WHERE id = $1;

-- name: LockEnvironment :one
-- Bloquea la fila hasta el fin de la transacción: dos escrituras sobre el mismo ambiente van una tras otra.
SELECT id
FROM environment
WHERE code = $1
    FOR UPDATE;

-- name: GetEnvironmentByCode :one
-- Lo instalado es lo que dejó el último despliegue exitoso del ambiente.
SELECT e.id, e.code, e.name, e.kind, e.os, e.notes, e.retired_at,
       e.created_by, u.display_name AS created_by_name, e.created_at, e.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       i.code AS installed_deployment, i.executed_at AS installed_at,
       ib.id AS installed_build_id, ib.code AS installed_build_code, ib.version AS installed_build_version,
       ib.checksum AS installed_build_checksum,
       ip.id AS installed_package_id, ip.code AS installed_package_code, ip.title AS installed_package_title
FROM environment e
         JOIN system_under_test s ON s.id = e.system_id
         JOIN app_user u ON u.id = e.created_by
         LEFT JOIN deployment i ON i.id = (SELECT d.id
                                           FROM deployment d
                                           WHERE d.environment_id = e.id
                                             AND d.result = 'success'
                                           ORDER BY d.executed_at DESC, d.created_at DESC, d.code DESC
                                           LIMIT 1)
         LEFT JOIN build ib ON ib.id = i.build_id
         LEFT JOIN package ip ON ip.id = i.package_id
WHERE e.code = $1;

-- name: ListEnvironments :many
SELECT e.id, e.code, e.name, e.kind, e.os, e.notes, e.retired_at,
       e.created_by, u.display_name AS created_by_name, e.created_at, e.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       i.code AS installed_deployment, i.executed_at AS installed_at,
       ib.id AS installed_build_id, ib.code AS installed_build_code, ib.version AS installed_build_version,
       ib.checksum AS installed_build_checksum,
       ip.id AS installed_package_id, ip.code AS installed_package_code, ip.title AS installed_package_title
FROM environment e
         JOIN system_under_test s ON s.id = e.system_id
         JOIN app_user u ON u.id = e.created_by
         LEFT JOIN deployment i ON i.id = (SELECT d.id
                                           FROM deployment d
                                           WHERE d.environment_id = e.id
                                             AND d.result = 'success'
                                           ORDER BY d.executed_at DESC, d.created_at DESC, d.code DESC
                                           LIMIT 1)
         LEFT JOIN build ib ON ib.id = i.build_id
         LEFT JOIN package ip ON ip.id = i.package_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(kind)::text IS NULL OR e.kind = sqlc.narg(kind))
  AND (sqlc.arg(include_retired)::boolean OR e.retired_at IS NULL)
ORDER BY s.key, e.name, e.code
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountEnvironments :one
SELECT count(*)
FROM environment e
         JOIN system_under_test s ON s.id = e.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(kind)::text IS NULL OR e.kind = sqlc.narg(kind))
  AND (sqlc.arg(include_retired)::boolean OR e.retired_at IS NULL);

-- name: GetPackageRef :one
-- Lee el PKG que se despliega: es el sentido de la llave foránea deployment.package_id.
SELECT p.id, p.code, p.title, s.key AS system_key
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
WHERE p.code = $1;

-- name: InsertDeployment :exec
INSERT INTO deployment (id, code, environment_id, build_id, package_id, result, notes, executed_by, executed_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetDeploymentByCode :one
SELECT d.id, d.code, d.result, d.notes, d.executed_by, u.display_name AS executed_by_name, d.executed_at, d.created_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       b.id AS build_id, b.code AS build_code, b.version AS build_version, b.checksum AS build_checksum,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM deployment d
         JOIN environment e ON e.id = d.environment_id
         JOIN app_user u ON u.id = d.executed_by
         LEFT JOIN build b ON b.id = d.build_id
         LEFT JOIN package p ON p.id = d.package_id
WHERE d.code = $1;

-- name: ListDeployments :many
SELECT d.id, d.code, d.result, d.notes, d.executed_by, u.display_name AS executed_by_name, d.executed_at, d.created_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       b.id AS build_id, b.code AS build_code, b.version AS build_version, b.checksum AS build_checksum,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM deployment d
         JOIN environment e ON e.id = d.environment_id
         JOIN app_user u ON u.id = d.executed_by
         LEFT JOIN build b ON b.id = d.build_id
         LEFT JOIN package p ON p.id = d.package_id
WHERE (sqlc.narg(environment_code)::text IS NULL OR e.code = sqlc.narg(environment_code))
  AND (sqlc.narg(build_code)::text IS NULL OR b.code = sqlc.narg(build_code))
  AND (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code))
ORDER BY d.executed_at DESC, d.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountDeployments :one
SELECT count(*)
FROM deployment d
         JOIN environment e ON e.id = d.environment_id
         LEFT JOIN build b ON b.id = d.build_id
         LEFT JOIN package p ON p.id = d.package_id
WHERE (sqlc.narg(environment_code)::text IS NULL OR e.code = sqlc.narg(environment_code))
  AND (sqlc.narg(build_code)::text IS NULL OR b.code = sqlc.narg(build_code))
  AND (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code));

-- name: HasProductionDeployment :one
SELECT EXISTS (SELECT 1
               FROM deployment d
                        JOIN environment e ON e.id = d.environment_id
                        JOIN package p ON p.id = d.package_id
               WHERE p.code = $1
                 AND d.result = 'success'
                 AND e.kind = 'production');
